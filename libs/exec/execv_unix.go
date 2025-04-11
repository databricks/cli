//go:build linux || darwin

package exec

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func execv(opts ExecvOptions) error {
	err := os.Chdir(opts.Dir)
	if err != nil {
		return fmt.Errorf("changing directory to %s failed: %w", opts.Dir, err)
	}

	// execve syscall does not perform PATH lookup and
	path, err := exec.LookPath(opts.Args[0])
	if err != nil {
		return fmt.Errorf("looking up %q failed: %w", opts.Args[0], err)
	}
	return syscall.Exec(path, opts.Args, opts.Env)
}
