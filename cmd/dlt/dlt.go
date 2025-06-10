package dlt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// InstallDLTSymlink creates a symlink named 'dlt' pointing to the real databricks binary.
func InstallDLTSymlink() error {
	path, err := exec.LookPath("databricks")
	if err != nil {
		return errors.New("databricks CLI not found in PATH")
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink: %w", err)
	}

	dir := filepath.Dir(path)
	dltPath := filepath.Join(dir, "dlt")

	// Check if 'dlt' already exists
	if fi, err := os.Lstat(dltPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(dltPath)
			if err == nil && target == realPath {
				cmdio.LogString(context.Background(), "dlt already installed")
				return nil
			}
		}
		return fmt.Errorf("cannot create symlink: %q already exists", dltPath)
	} else if !os.IsNotExist(err) {
		// Some other error occurred while checking
		return fmt.Errorf("failed to check if %q exists: %w", dltPath, err)
	}

	if err := os.Symlink(realPath, dltPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	return nil
}

func New() *cobra.Command {
	return &cobra.Command{
		Use:    "install-dlt",
		Short:  "Install DLT",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return InstallDLTSymlink()
		},
	}
}

func NewRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dlt",
		Short: "DLT CLI",
		Long:  "DLT CLI (stub, to be filled in)",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add 'init' stub command (same description as bundle init)
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new DLT project in the current directory",
		Long:  "Initialize a new DLT project in the current directory. This is a stub for future implementation.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("dlt init is not yet implemented. This will initialize a new DLT project in the future.")
		},
	}
	cmd.AddCommand(initCmd)

	return cmd
}
