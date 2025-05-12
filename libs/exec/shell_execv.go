package exec

import "errors"

// Variant of [Execv] that runs the given script through a shell
func ShellExecv(content string, opts ExecvOptions) error {
	if opts.Args != nil {
		return errors.New("ShellExecv: Args is not supported")
	}

	shell, err := findShell()
	if err != nil {
		return err
	}

	ec, err := shell.prepare(content)
	if err != nil {
		return err
	}

	args := []string{ec.executable}
	args = append(args, ec.args...)

	return Execv(ExecvOptions{
		Args: args,
		Env:  opts.Env,
		Dir:  opts.Dir,
	})
}
