package exec

import (
	"errors"
	"io"
	"os"
)

type shell interface {
	prepare(string) (*execContext, error)
	getType() ExecutableType
}

type execContext struct {
	executable string
	args       []string
	scriptFile string
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

func createTempScript(command string, extension string) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), "cli-exec*"+extension)
	if err != nil {
		return "", err
	}

	defer file.Close()

	_, err = io.WriteString(file, command)
	if err != nil {
		// Try to remove the file if we failed to write to it
		os.Remove(file.Name())
		return "", err
	}

	return file.Name(), nil
}
