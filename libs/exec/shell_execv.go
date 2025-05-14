package exec

import "errors"

func shellExecvOpts(content string, opts ExecvOptions) (ExecvOptions, error) {
	if opts.Args != nil {
		return ExecvOptions{}, errors.New("ShellExecv: Args is not supported")
	}

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
		Env:  opts.Env,
		Dir:  opts.Dir,
	}, nil
}

// Variant of [Execv] that runs the given script through a shell
func ShellExecv(content string, opts ExecvOptions) error {
	newOpts, err := shellExecvOpts(content, opts)
	if err != nil {
		return err
	}

	return Execv(newOpts)
}
