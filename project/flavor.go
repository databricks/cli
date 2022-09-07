package project

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type Flavor interface {
	// Name returns a tuple of flavor key and readable name
	Name() (string, string)

	// Detected returns true on successful metadata checks
	Detected() bool

	// Build triggers packaging subprocesses
	Build(context.Context) error
	// TODO: Init() Questions
	// TODO: Deploy(context.Context) error
}

var _ Flavor = PythonWheel{}

type PythonWheel struct{}

func (pw PythonWheel) Name() (string, string) {
	return "wheel", "Python Wheel"
}

func (pw PythonWheel) Detected() bool {
	root, err := findProjectRoot()
	if err != nil {
		return false
	}
	_, err = os.Stat(fmt.Sprintf("%s/setup.py", root))
	return err == nil
}

func (pw PythonWheel) Build(ctx context.Context) error {
	defer toTheRootAndBack()()
	// do subprocesses or https://github.com/go-python/cpy3
	// it all depends on complexity and binary size
	// TODO: detect if there's an .venv here and call setup.py with ENV vars of it
	// TODO: where.exe python (WIN) / which python (UNIX)
	cmd := exec.CommandContext(ctx, "python", "setup.py", "bdist-wheel")
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func toTheRootAndBack() func() {
	wd, _ := os.Getwd()
	root, _ := findProjectRoot()
	os.Chdir(root)
	return func() {
		os.Chdir(wd)
	}
}

var _ Flavor = PythonNotebooks{}

type PythonNotebooks struct{}

func (n PythonNotebooks) Name() (string, string) {
	// or just "notebooks", as we might shuffle in scala?...
	return "python-notebooks", "Python Notebooks"
}

func (n PythonNotebooks) Detected() bool {
	// TODO: Steps:
	// - get all filenames
	// - read first X bytes from random 10 files and check
	// if they're "Databricks Notebook Source"
	return false
}

func (n PythonNotebooks) Build(ctx context.Context) error {
	// TODO: perhaps some linting?..
	return nil
}

func (n PythonNotebooks) Deploy(ctx context.Context) error {
	// TODO: recursively upload notebooks to a given workspace path
	return nil
}
