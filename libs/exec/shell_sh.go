package exec

import (
	"errors"
	osexec "os/exec"
)

type shShell struct {
	executable string
}

func (s shShell) prepare(command string) (*execContext, error) {
	filename, err := createTempScript(command, ".sh")
	if err != nil {
		return nil, err
	}

	return &execContext{
		executable: s.executable,
		args:       []string{"-e", filename},
		scriptFile: filename,
	}, nil
}

func newShShell() (shell, error) {
	out, err := osexec.LookPath("sh")
	if err != nil && !errors.Is(err, osexec.ErrNotFound) {
		return nil, err
	}

	// `sh` is not found, return early.
	if out == "" {
		return nil, nil
	}

	return &shShell{executable: out}, nil
}
