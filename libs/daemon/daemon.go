package daemon

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
)

type Daemon struct {
	// If provided, the child process's pid will be written in the file at this
	// path.
	PidFilePath string

	// Environment variables to set in the child process.
	Env []string

	// Path to executable to run. If empty, the current executable is used.
	Executable string

	// Arguments to pass to the child process.
	Args []string

	// Log file to write the child process's output to.
	LogFile string

	logFile *os.File
	cmd     *exec.Cmd
	stdin   io.WriteCloser
}

func (d *Daemon) Start() error {
	var err error
	executable := d.Executable
	if executable == "" {
		// If Executable is not provided, use the current CLI executable.
		executable, err = os.Executable()
		if err != nil {
			return err
		}
	}

	d.cmd = exec.Command(executable, d.Args...)
	d.cmd.Env = d.Env

	d.cmd.SysProcAttr = sysProcAttr()

	// By default redirect stdout and stderr to /dev/null.
	d.cmd.Stdout = nil
	d.cmd.Stderr = nil

	// If a log file is provided, redirect stdout and stderr to the log file.
	if d.LogFile != "" {
		d.logFile, err = os.OpenFile(d.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		// The file descriptor for the log file is closed in the [Daemon.Release] method.
		d.cmd.Stdout = d.logFile
		d.cmd.Stderr = d.logFile
	}

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

func (d *Daemon) WriteInput(b []byte) error {
	_, err := d.stdin.Write(b)
	return err
}

func (d *Daemon) Release() error {
	if d.stdin != nil {
		err := d.stdin.Close()
		if err != nil {
			return fmt.Errorf("failed to close stdin: %w", err)
		}
	}

	// Note that the child process will stream its output directly to the log file.
	// So it's safe to close this file handle even if the child process is still running.
	if d.logFile != nil {
		err := d.logFile.Close()
		if err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
	}

	if d.cmd == nil {
		return nil
	}

	// The docs for [os.Process.Release] specify that we need to call Release if
	// Wait is not called.
	return d.cmd.Process.Release()
}
