package exec

import (
	"errors"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
)

type interpreter interface {
	prepare(string) (*execContext, error)
	cleanup(*execContext) error
}

type execContext struct {
	executable string
	args       []string
	scriptFile string
}

type bashInterpreter struct {
	executable string
}

func (b *bashInterpreter) prepare(command string) (*execContext, error) {
	filename, err := createTempScript(command, ".sh")
	if err != nil {
		return nil, err
	}

	return &execContext{
		executable: b.executable,
		args:       []string{"-e", filename},
		scriptFile: filename,
	}, nil
}

func (b *bashInterpreter) cleanup(ec *execContext) error {
	return os.Remove(ec.scriptFile)
}

type cmdInterpreter struct {
	executable string
}

func (c *cmdInterpreter) prepare(command string) (*execContext, error) {
	filename, err := createTempScript(command, ".cmd")
	if err != nil {
		return nil, err
	}

	return &execContext{
		executable: c.executable,
		args:       []string{"/D", "/E:ON", "/V:OFF", "/S", "/C", fmt.Sprintf(`CALL %s`, filename)},
		scriptFile: filename,
	}, nil
}

func (c *cmdInterpreter) cleanup(ec *execContext) error {
	return os.Remove(ec.scriptFile)
}

func findInterpreter() (interpreter, error) {
	interpreter, err := findBashInterpreter()
	if err != nil {
		return nil, err
	}

	if interpreter != nil {
		return interpreter, nil
	}

	interpreter, err = findCmdInterpreter()
	if err != nil {
		return nil, err
	}

	if interpreter != nil {
		return interpreter, nil
	}

	return nil, errors.New("no interpreter found")
}

func findBashInterpreter() (interpreter, error) {
	// Lookup for bash executable first (Linux, MacOS, maybe Windows)
	out, err := osexec.LookPath("bash")
	if err != nil && !errors.Is(err, osexec.ErrNotFound) {
		return nil, err
	}

	// Bash executable is not found, returning early
	if out == "" {
		return nil, nil
	}

	return &bashInterpreter{executable: out}, nil
}

func findCmdInterpreter() (interpreter, error) {
	// Lookup for CMD executable (Windows)
	out, err := osexec.LookPath("cmd")
	if err != nil && !errors.Is(err, osexec.ErrNotFound) {
		return nil, err
	}

	// CMD executable is not found, returning early
	if out == "" {
		return nil, nil
	}

	return &cmdInterpreter{executable: out}, nil
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
