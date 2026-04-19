//go:build linux || darwin

package acceptance_test

import (
	"os/exec"
	"syscall"
)

// setProcessGroup puts cmd into a new process group so that killTree can
// terminate all of its descendants at once when a test script times out.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killTree SIGKILLs the process group rooted at cmd. Without this, a timed-out
// bash script is killed but its children (databricks, terraform) are reparented
// to init and continue running, consuming resources across subsequent tests.
func killTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
