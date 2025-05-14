package exec

import (
	"errors"
	"io"
)

type shell interface {
	prepare(string) (*execContext, error)
	getType() ExecutableType
}

type execContext struct {
	executable string
	args       []string
	stdin      io.Reader
}

func findShell() (shell, error) {
	for _, fn := range []func() (shell, error){
		newBashShell,
		newShShell,
		newCmdShell,
	} {
		shell, err := fn()
		if err != nil {
			return nil, err
		}

		if shell != nil {
			return shell, nil
		}
	}

	return nil, errors.New("no shell found")
}
