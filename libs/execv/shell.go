package execv

import "github.com/databricks/cli/libs/exec"

// Shell executes a shell script by replacing the current process.
//
// The script content is executed through the system's default shell:
// - On Unix systems: bash (preferred) or sh
// - On Windows: cmd.exe
//
// This function calls the execve syscall on Unix systems, which replaces
// the current process. On Windows, it creates a child process and exits
// with the child's exit code.
//
// Note: For cmd.exe, the script is written to a temporary file which is
// automatically cleaned up. On Unix systems, temporary files in /tmp are
// periodically cleaned up by the system.
//
// This function does not return on success.
func Shell(content, dir string, env []string) error {
	prep, err := exec.PrepareShellCommand(content)
	if err != nil {
		return err
	}

	return Execv(Options{
		Args:    prep.Args,
		Env:     env,
		Dir:     dir,
		cleanup: prep.CleanupFn,
	})
}
