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
	}, nil
}

// Variant of [Execv] that runs the given script through a shell
func ShellExecv(content, dir string, env []string) error {
	newOpts, err := shellExecvOpts(content, dir, env)
	if err != nil {
		return err
	}

	return Execv(newOpts)
}
