package dlt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func InstallDLTSymlink(directory string) error {
	path, err := os.Executable()
	if err != nil {
		return errors.New("databricks CLI executable not found")
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink: %w", err)
	}

	dir := directory
	if dir == "" {
		dir = filepath.Dir(path)
	}
	dltPath := filepath.Join(dir, "dlt")

	if fi, err := os.Lstat(dltPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(dltPath)
			if err == nil && target != realPath {
				return fmt.Errorf("cannot install dlt CLI: %q already exists", dltPath)
			}
			if err != nil {
				return err
			}
		}
	} else if os.IsNotExist(err) {
		if err := os.Symlink(realPath, dltPath); err != nil {
			return fmt.Errorf("failed to install dlt CLI: %w", err)
		}
	} else {
		return fmt.Errorf("failed to check if %q exists: %w", dltPath, err)
	}
	cmdio.LogString(context.Background(), fmt.Sprintf("dlt successfully installed to the directory %q", dir))
	return nil
}

func InstallDLT() *cobra.Command {
	var directory string
	cmd := &cobra.Command{
		Use:    "install-dlt",
		Short:  "Install DLT",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return InstallDLTSymlink(directory)
		},
	}
	cmd.Flags().StringVarP(&directory, "directory", "d", "", "Directory in which to install dlt CLI (defaults to databricks CLI's directory)")
	return cmd
}
