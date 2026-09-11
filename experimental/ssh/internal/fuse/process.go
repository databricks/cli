package fuse

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Registration identifies a process without allowing a recycled PID to inherit its credentials.
type Registration struct {
	PID            int
	PIDNamespaceID uint32
	StartTime      uint64
}

// Self describes the calling process using Linux procfs.
func Self() (Registration, error) {
	link, err := os.Readlink("/proc/self/ns/pid")
	if err != nil {
		return Registration{}, fmt.Errorf("failed to read the PID namespace: %w", err)
	}
	stat, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return Registration{}, fmt.Errorf("failed to read process stat: %w", err)
	}
	return parseRegistration(os.Getpid(), link, string(stat))
}

func parseRegistration(pid int, namespace, stat string) (Registration, error) {
	inner, ok := strings.CutPrefix(namespace, "pid:[")
	if !ok {
		return Registration{}, fmt.Errorf("unexpected PID namespace link %q", namespace)
	}
	inner, ok = strings.CutSuffix(inner, "]")
	if !ok {
		return Registration{}, fmt.Errorf("unexpected PID namespace link %q", namespace)
	}
	namespaceID, err := strconv.ParseUint(inner, 10, 32)
	if err != nil {
		return Registration{}, fmt.Errorf("failed to parse the PID namespace: %w", err)
	}

	// Field 2 can contain spaces and parentheses. Count field 22 from the last ')'.
	// https://man7.org/linux/man-pages/man5/proc_pid_stat.5.html
	end := strings.LastIndex(stat, ")")
	if end < 0 {
		return Registration{}, errors.New("missing process name in process stat")
	}
	fields := strings.Fields(stat[end+1:])
	const startTimeIndex = 19
	if len(fields) <= startTimeIndex {
		return Registration{}, errors.New("missing start time in process stat")
	}
	startTime, err := strconv.ParseUint(fields[startTimeIndex], 10, 64)
	if err != nil {
		return Registration{}, fmt.Errorf("failed to parse process start time: %w", err)
	}
	return Registration{PID: pid, PIDNamespaceID: uint32(namespaceID), StartTime: startTime}, nil
}
