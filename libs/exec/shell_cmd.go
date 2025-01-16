package exec

import (
	"errors"
	osexec "os/exec"
)

type cmdShell struct {
	executable string
}

func (s cmdShell) prepare(command string) (*execContext, error) {
	filename, err := createTempScript(command, ".cmd")
	if err != nil {
		return nil, err
	}

	return &execContext{
		executable: s.executable,
		args:       []string{"/D", "/E:ON", "/V:OFF", "/S", "/C", "CALL " + filename},
		scriptFile: filename,
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
