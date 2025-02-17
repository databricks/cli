package process

import "fmt"

type ErrProcessNotFound struct {
	Pid int
}

func (e ErrProcessNotFound) Error() string {
	return fmt.Sprintf("process with pid %d does not exist", e.Pid)
}

func Wait(pid int) error {
	return waitForPid(pid)
}
