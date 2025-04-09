//go:build windows

package exec

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Note: Windows does not support an execv syscall that replaces the current process.
// Thus we emulate that by launching a child process and streaming the output and returning
// the exit code.
// ref: https://learn.microsoft.com/en-us/cpp/c-runtime-library/reference/execv-wexecv?view=msvc-170
func execv(opts ExecvOptions) error {
	cmd := exec.Command(opts.Name, opts.Args...)

	// TODO: Move this comment.
	// Execute all scripts from the bundle root directory. This behavior can
	// be surprising in isolation, but we do it to keep the behavior consistent
	// for both these cases:
	// 1. One shot commands like `databricks bundle exec -- echo hello`
	// 2. (upcoming) Scripts that are defined in the scripts section of the DAB.
	cmd.Dir = opts.Dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe failed: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe failed: %w", err)
	}

	// Start the child command.
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("starting %s %s failed: %w", opts.Name, strings.Join(opts.Args, " "), err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var stdoutErr error
	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			_, err = fmt.Fprintln(opts.Stdout, scanner.Text())
			if err != nil {
				stdoutErr = err
				break
			}
		}
	}()

	var stderrErr error
	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			_, err = fmt.Fprintln(opts.Stderr, scanner.Text())
			if err != nil {
				stderrErr = err
				break
			}
		}
	}()

	wg.Wait()

	if stdoutErr != nil {
		return fmt.Errorf("writing stdout failed: %w", stdoutErr)
	}

	if stderrErr != nil {
		return fmt.Errorf("writing stderr failed: %w", stderrErr)
	}

	err = cmd.Wait()
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}
	if err != nil {
		return fmt.Errorf("running %s %s failed: %w", opts.Name, strings.Join(opts.Args, " "), err)
	}

	return nil
}
