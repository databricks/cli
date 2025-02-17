//go:build linux || darwin

package process

import (
	"errors"
	"os"
	"syscall"
	"time"
)

func waitForPid(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	// Initial existence check.
	if err := p.Signal(syscall.Signal(0)); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return ErrProcessNotFound{Pid: pid}
		}
		return err
	}

	// Polling loop until process exits
	for {
		if err := p.Signal(syscall.Signal(0)); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				return nil
			}
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
}
