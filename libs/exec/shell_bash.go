package exec

import (
	"errors"
	osexec "os/exec"
	"strings"
)

type bashShell struct {
	executable string
}

func (s bashShell) prepare(command string) (*execContext, error) {
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

func newBashShell() (shell, error) {
	out, err := osexec.LookPath("bash")
	if err != nil && !errors.Is(err, osexec.ErrNotFound) {
		return nil, err
	}

	// `bash` is not found, return early.
	if out == "" {
		return nil, nil
	}

	// Skipping WSL bash if found one
	if strings.Contains(out, `\Windows\System32\bash.exe`) || strings.Contains(out, `\Microsoft\WindowsApps\bash.exe`) {
		return nil, nil
	}

	return &bashShell{executable: out}, nil
}
