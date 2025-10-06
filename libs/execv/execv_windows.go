//go:build windows

package execv

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// execv emulates Unix execv behavior on Windows.
//
// Windows does not support an execv syscall that replaces the current process.
// To emulate this, we create a child process, pass the stdin, stdout and stderr
// file descriptors, wait for it to complete, and exit with its exit code.
//
// ref: https://learn.microsoft.com/en-us/cpp/c-runtime-library/reference/execv-wexecv?view=msvc-170
func execv(opts Options) error {
	if opts.cleanup != nil {
		defer opts.cleanup()
	}

	windowsExit := func(status int) {
		// Clean up before exiting (defer will not run).
		if opts.cleanup != nil {
			opts.cleanup()
		}
		opts.windowsExit(status)
	}

	path, err := exec.LookPath(opts.Args[0])
	if err != nil {
		return fmt.Errorf("looking up %q failed: %w", opts.Args[0], err)
	}
	cmd := exec.Command(path, opts.Args[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Dir = opts.Dir
	cmd.Env = opts.Env

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("%s failed: %w", strings.Join(opts.Args, " "), err)
	}

	err = cmd.Wait()
	if exitErr, ok := err.(*exec.ExitError); ok {
		windowsExit(exitErr.ExitCode())
		return nil
	}
	if err != nil {
		return fmt.Errorf("running %s failed: %w", strings.Join(opts.Args, " "), err)
	}

	// Unix implementation of execv never returns control to the CLI process.
	// To emulate this behavior, we exit early here if the child process exits
	// successfully.
	windowsExit(0)
	return nil
}
