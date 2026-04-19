//go:build windows

package acceptance_test

import "os/exec"

func setProcessGroup(cmd *exec.Cmd) {}

func killTree(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
