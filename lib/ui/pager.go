package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func startPager(cmd *cobra.Command) (func() error, error) {
	pagerFromEnv := os.Getenv("PAGER")
	if !Interactive || filepath.Base(pagerFromEnv) == "cat" || !isTTY(cmd.OutOrStdout()) {
		return func() error { return nil }, nil
	}
	var lessEnvSet bool
	passEnv := []string{}
	for _, line := range os.Environ() {
		if strings.HasPrefix(line, "PAGER=") {
			continue
		}
		if strings.HasPrefix(line, "LESS=") {
			lessEnvSet = true
		}
		passEnv = append(passEnv, line)
	}
	if !lessEnvSet {
		// -F: Causes less to automatically exit if the entire file can be displayed on the first screen.
		// -R: only ANSI "color" escape sequences are output in "raw" form. The  screen appearance is
		//     maintained correctly in most cases.
		// -X: Disables sending the termcap initialization and deinitialization strings to the terminal. This is
		//     sometimes desirable if the deinitialization string does something unnecessary, like clearing the
		//     screen.
		passEnv = append(passEnv, "LESS=FRX")
		passEnv = append(passEnv, "LV=-c")
	}
	pager := exec.Command(pagerFromEnv)
	pager.Env = passEnv
	pager.Stdout = cmd.OutOrStdout()
	pager.Stderr = cmd.ErrOrStderr()
	pagedOut, err := pager.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	cmd.SetOut(pagedOut) // TODO: exiting main with OK for syscall.Errno(232) on windows and syscall.EPIPE
	err = pager.Start()
	if err != nil {
		return nil, fmt.Errorf("start pager: %w", err)
	}
	return func() error {
		// if a pager was started, we're guaranteed to have a WriteCloser
		cmd.OutOrStdout().(io.WriteCloser).Close()
		pager.Process.Wait()
		pager.Process = nil
		return nil
	}, nil
}
