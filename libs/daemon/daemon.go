package daemon

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
)

type Daemon struct {
	// If provided, the child process will create a pid file at this path.
	PidFilePath string

	// Environment variables to set in the child process.
	Env []string

	// Arguments to pass to the child process.
	Args []string

	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func (d *Daemon) Start() error {
	cli, err := os.Executable()
	if err != nil {
		return err
	}

	d.cmd = exec.Command(cli, d.Args...)
	d.cmd.Env = d.Env
	d.cmd.SysProcAttr = sysProcAttr()

	d.stdin, err = d.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	err = d.cmd.Start()
	if err != nil {
		return err
	}

	if d.PidFilePath != "" {
		err = os.WriteFile(d.PidFilePath, []byte(strconv.Itoa(d.cmd.Process.Pid)), 0o644)
		if err != nil {
			return fmt.Errorf("failed to write pid file: %w", err)
		}
	}

	return nil
}

func (d *Daemon) Release() error {
	if d.PidFilePath != "" {
		err := os.Remove(d.PidFilePath)
		if err != nil {
			return fmt.Errorf("failed to remove pid file: %w", err)
		}
	}

	if d.stdin != nil {
		err := d.stdin.Close()
		if err != nil {
			return fmt.Errorf("failed to close stdin: %w", err)
		}
	}

	if d.cmd == nil {
		return nil
	}

	return d.cmd.Process.Release()
}
