package exec

func shellExecvOpts(content, dir string, env []string) (ExecvOptions, error) {
	shell, err := findShell()
	if err != nil {
		return ExecvOptions{}, err
	}

	ec, err := shell.prepare(content)
	if err != nil {
		return ExecvOptions{}, err
	}

	args := []string{ec.executable}
	args = append(args, ec.args...)

	return ExecvOptions{
		Args: args,
		Env:  env,
		Dir:  dir,
		cleanup: func() {
			ec.cleanup()
		},
	}, nil
}

// Variant of [Execv] that runs the given script through a shell
func ShellExecv(content, dir string, env []string) error {
	opts, err := shellExecvOpts(content, dir, env)
	if err != nil {
		return err
	}

	// Note: To execute the content of the script, we write a temporary file and execute that
	// using a shell. Since execv does not return control to the CLI process, we need to
	// clean up the temporary file here.
	// This is fine for Unix and Mac OS systems because files in /tmp are automatically
	// cleaned up.
	// ref1: https://serverfault.com/questions/377348/when-does-tmp-get-cleared
	// ref2: https://superuser.com/questions/187071/in-macos-how-often-is-tmp-deleted
	//
	// For windows the temp files are not automatically cleaned up. Automatic cleanup
	// is a opt-in behavior.
	return Execv(opts)
}
