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
	return &execContext{
		executable: s.executable,
		args:       []string{"-ec", command},
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

	// Convert to lowercase for case-insensitive comparison
	// Some systems may return some parts of the path in uppercase.
	outLower := strings.ToLower(out)
	// Skipping WSL bash if found one
	if strings.Contains(outLower, `\windows\system32\bash.exe`) ||
		strings.Contains(outLower, `\microsoft\windowsapps\bash.exe`) {
		return nil, nil
	}

	return &bashShell{executable: out}, nil
}

func (s bashShell) getType() ExecutableType {
	return BashExecutable
}
