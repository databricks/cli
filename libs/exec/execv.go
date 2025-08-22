package exec

import "os"

type ExecvOptions struct {
	// Args is the name of the command to run and its arguments.
	// Eg: ["echo", "hello"] for "echo hello"
	Args []string

	// Env is set the environment variables to set in the child process.
	Env []string

	// Dir specifies the working directory of the command.
	// If Dir is an empty string, Execv runs the command in the
	// calling process's current directory.
	Dir string

	// It is not possible to execute a cmd.exe script inlined as a argument
	// to cmd.exe. They have to be serialized to a file and executed.
	// Thus if [Execv] is used to execution a script though cmd.exe,
	// the caller should register a cleanup function to clean up the temporary file.
	cleanup func()

	// Callback to exit the current process in windows. Having this as a function here
	// helps with testing.
	windowsExit func(status int)
}

func Execv(opts ExecvOptions) error {
	if opts.windowsExit == nil {
		opts.windowsExit = os.Exit
	}
	return execv(opts)
}
