//go:build !linux && !windows

package detector

import "runtime"

func platformProcesses() ([]processCandidate, error) {
	return nil, unsupportedPlatformError(runtime.GOOS)
}

func platformLoopbackPorts(_ int, _ []int) ([]int, error) {
	return nil, unsupportedPlatformError(runtime.GOOS)
}
