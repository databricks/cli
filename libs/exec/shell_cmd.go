package exec

import (
	"bytes"
	"errors"
	osexec "os/exec"
)

type cmdShell struct {
	executable string
}

func (s cmdShell) prepare(command string) (*execContext, error) {
	reader := bytes.NewReader([]byte(command))

	return &execContext{
		executable: s.executable,
		args:       []string{"/D", "/E:ON", "/V:OFF"},
		stdin:      reader,
	}, nil
}

func newCmdShell() (shell, error) {
	out, err := osexec.LookPath("cmd")
	if err != nil && !errors.Is(err, osexec.ErrNotFound) {
		return nil, err
	}

	// `cmd.exe` is not found, return early.
	if out == "" {
		return nil, nil
	}

	return &cmdShell{executable: out}, nil
}

func (s cmdShell) getType() ExecutableType {
	return CmdExecutable
}
