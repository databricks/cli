package exec

type ExecvOptions struct {
	// Args is the name of the command to run and its arguments.
	// Eg: ["echo", "hello"] for "echo hello"
	Args []string

	// Env is set the environment variables to set in the child process.
	Env []string

	// Dir is the working directory of the child process.
	Dir string

	WindowsCleanupTempScript func()
}

func Execv(opts ExecvOptions) error {
	return execv(opts)
}
