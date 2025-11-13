package exec

// ShellPreparation contains everything needed to execute a shell command.
type ShellPreparation struct {
	// Executable is the path to the shell executable.
	Executable string
	// Args contains all arguments including the executable as the first element.
	Args []string
	// CleanupFn should be called to clean up any temporary resources.
	// It may be nil if no cleanup is needed.
	CleanupFn func()
}

// PrepareShellCommand prepares a command string for execution through a shell.
// It finds an available shell (bash, sh, or cmd.exe) and returns the prepared
// command with executable, arguments, and an optional cleanup function.
func PrepareShellCommand(command string) (*ShellPreparation, error) {
	shell, err := findShell()
	if err != nil {
		return nil, err
	}

	ec, err := shell.prepare(command)
	if err != nil {
		return nil, err
	}

	args := []string{ec.executable}
	args = append(args, ec.args...)

	var cleanupFn func()
	if ec.scriptFile != "" {
		cleanupFn = func() {
			ec.cleanup()
		}
	}

	return &ShellPreparation{
		Executable: ec.executable,
		Args:       args,
		CleanupFn:  cleanupFn,
	}, nil
}
