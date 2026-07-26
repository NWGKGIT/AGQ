//go:build windows

package detector

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const windowsTCPTableOwnerPIDListener = 3

var getExtendedTCPTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

type nativeUnicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}

func platformProcesses() ([]processCandidate, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("create process snapshot: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, fmt.Errorf("read first process: %w", err)
	}

	var candidates []processCandidate
	for {
		executable := windows.UTF16ToString(entry.ExeFile[:])
		if hasLanguageServerExecutable([]string{executable}) {
			args, err := windowsProcessArgs(entry.ProcessID)
			if err == nil {
				candidates = append(candidates, processCandidate{pid: int(entry.ProcessID), args: args})
			}
		}

		err := windows.Process32Next(snapshot, &entry)
		if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read next process: %w", err)
		}
	}
	return candidates, nil
}

func windowsProcessArgs(pid uint32) ([]string, error) {
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return nil, fmt.Errorf("open process %d: %w", pid, err)
	}
	defer windows.CloseHandle(process)

	var size uint32
	queryErr := windows.NtQueryInformationProcess(
		process,
		windows.ProcessCommandLineInformation,
		nil,
		0,
		&size,
	)
	if size == 0 {
		return nil, fmt.Errorf("query process %d command line size: %w", pid, queryErr)
	}

	buffer := make([]byte, size)
	if err := windows.NtQueryInformationProcess(
		process,
		windows.ProcessCommandLineInformation,
		unsafe.Pointer(&buffer[0]),
		uint32(len(buffer)),
		&size,
	); err != nil {
		return nil, fmt.Errorf("query process %d command line: %w", pid, err)
	}

	commandLine, err := commandLineFromNativeBuffer(buffer)
	if err != nil {
		return nil, fmt.Errorf("decode process %d command line: %w", pid, err)
	}
	args, err := windows.DecomposeCommandLine(commandLine)
	if err != nil {
		return nil, fmt.Errorf("split process %d command line: %w", pid, err)
	}
	return args, nil
}

func commandLineFromNativeBuffer(buffer []byte) (string, error) {
	if len(buffer) < int(unsafe.Sizeof(nativeUnicodeString{})) {
		return "", fmt.Errorf("command line buffer is too short")
	}
	native := (*nativeUnicodeString)(unsafe.Pointer(&buffer[0]))
	if native.Length == 0 {
		return "", nil
	}
	if native.Buffer == nil || native.Length%2 != 0 || native.Length > native.MaximumLength {
		return "", fmt.Errorf("invalid UNICODE_STRING")
	}
	bufferStart := uintptr(unsafe.Pointer(&buffer[0]))
	bufferEnd := bufferStart + uintptr(len(buffer))
	stringStart := uintptr(unsafe.Pointer(native.Buffer))
	stringEnd := stringStart + uintptr(native.Length)
	if stringStart < bufferStart || stringEnd < stringStart || stringEnd > bufferEnd {
		return "", fmt.Errorf("UNICODE_STRING points outside its result buffer")
	}
	return windows.UTF16ToString(unsafe.Slice(native.Buffer, int(native.Length/2))), nil
}

func platformLoopbackPorts(pid int, declaredPorts []int) ([]int, error) {
	var ports []int
	var queryErrors []error
	for _, family := range []uint32{windowsAFInet, windowsAFInet6} {
		table, err := queryWindowsTCPTable(family)
		if err != nil {
			queryErrors = append(queryErrors, err)
			continue
		}
		ports = append(ports, parseWindowsTCPTable(table, family, pid)...)
	}
	ports = validPorts(ports)
	if len(ports) > 0 {
		return ports, errors.Join(queryErrors...)
	}

	// Antigravity normally declares its server ports in the command line.
	// Use those only when Windows has no PID-owned loopback listener to offer;
	// the PortProber still validates each candidate against 127.0.0.1.
	return validPorts(declaredPorts), errors.Join(queryErrors...)
}

func queryWindowsTCPTable(family uint32) ([]byte, error) {
	var size uint32
	result, _, _ := getExtendedTCPTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		uintptr(family),
		windowsTCPTableOwnerPIDListener,
		0,
	)
	if syscall.Errno(result) != windows.ERROR_INSUFFICIENT_BUFFER || size == 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable(%d) size: %w", family, syscall.Errno(result))
	}

	buffer := make([]byte, size)
	result, _, _ = getExtendedTCPTable.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		uintptr(family),
		windowsTCPTableOwnerPIDListener,
		0,
	)
	if result != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable(%d): %w", family, syscall.Errno(result))
	}
	return buffer[:size], nil
}
