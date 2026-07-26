package detector

import "encoding/binary"

const (
	windowsAFInet    = 2
	windowsAFInet6   = 23
	windowsTCPListen = 2

	windowsTCPv4OwnerPIDRowSize = 24
	windowsTCPv6OwnerPIDRowSize = 56
)

// parseWindowsTCPTable parses a MIB_TCPTABLE_OWNER_PID or
// MIB_TCP6TABLE_OWNER_PID buffer returned by GetExtendedTcpTable. Keeping the
// parser platform-neutral lets its filtering and bounds handling be tested on
// non-Windows builders.
func parseWindowsTCPTable(table []byte, family uint32, pid int) []int {
	if len(table) < 4 || pid <= 0 {
		return nil
	}

	rowSize := 0
	switch family {
	case windowsAFInet:
		rowSize = windowsTCPv4OwnerPIDRowSize
	case windowsAFInet6:
		rowSize = windowsTCPv6OwnerPIDRowSize
	default:
		return nil
	}

	count := int(binary.LittleEndian.Uint32(table[:4]))
	maxRows := (len(table) - 4) / rowSize
	if maxRows == 0 {
		return nil
	}
	if count > maxRows {
		count = maxRows
	}

	ports := make([]int, 0)
	for i := 0; i < count; i++ {
		row := table[4+i*rowSize : 4+(i+1)*rowSize]
		var stateOffset, pidOffset, portOffset int
		var loopback bool
		if family == windowsAFInet {
			stateOffset, pidOffset, portOffset = 0, 20, 8
			loopback = row[4] == 127 && row[5] == 0 && row[6] == 0 && row[7] == 1
		} else {
			stateOffset, pidOffset, portOffset = 48, 52, 20
			loopback = isIPv6Loopback(row[:16])
		}

		if binary.LittleEndian.Uint32(row[stateOffset:stateOffset+4]) != windowsTCPListen ||
			int(binary.LittleEndian.Uint32(row[pidOffset:pidOffset+4])) != pid ||
			!loopback {
			continue
		}

		// The port occupies a DWORD, but its first two bytes are in network
		// byte order according to the MIB_*ROW_OWNER_PID contract.
		port := int(binary.BigEndian.Uint16(row[portOffset : portOffset+2]))
		if port > 0 {
			ports = append(ports, port)
		}
	}
	return validPorts(ports)
}

func isIPv6Loopback(addr []byte) bool {
	if len(addr) != 16 || addr[15] != 1 {
		return false
	}
	for _, b := range addr[:15] {
		if b != 0 {
			return false
		}
	}
	return true
}
