package daemon

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
)

const DatabricksCliParentPid = "DATABRICKS_CLI_PARENT_PID"

type Daemon struct {
	// If provided, the child process will create a pid file at this path.
	PidFilePath string

	// Environment variables to set in the child process.
	Env []string

	// Path to executable to run. If empty, the current executable is used.
	Executable string

	// Arguments to pass to the child process.
	Args []string

	// Log file to write the child process's output to.
	LogFile string

	outFile *os.File
	cmd     *exec.Cmd
	stdin   io.WriteCloser
}

func (d *Daemon) Start() error {
	cli, err := os.Executable()
	if err != nil {
		return err
	}

	executable := d.Executable
	if executable == "" {
		executable = cli
	}

	d.cmd = exec.Command(executable, d.Args...)

	// Set environment variable so that the child process knows it's parent's PID.
	// In unix systems orphaned processes are automatically re-parented to init (pid 1)
	// so we cannot rely on os.Getppid() to get the original parent's pid.
	d.Env = append(d.Env, fmt.Sprintf("%s=%d", DatabricksCliParentPid, os.Getpid()))
	d.cmd.Env = d.Env

	d.cmd.SysProcAttr = sysProcAttr()

	// By default redirect stdout and stderr to /dev/null.
	d.cmd.Stdout = nil
	d.cmd.Stderr = nil

	// If a log file is provided, redirect stdout and stderr to the log file.
	if d.LogFile != "" {
		d.outFile, err = os.OpenFile(d.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		d.cmd.Stdout = d.outFile
		d.cmd.Stderr = d.outFile
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

	if d.outFile != nil {
		err := d.outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
	}

	if d.cmd == nil {
		return nil
	}

	// The docs recommend calling Release if Wait is not called. This does not
	// seem to be necessary since the functionality works without it. However, we
	// we add it just to be safe.
	return d.cmd.Process.Release()
}
