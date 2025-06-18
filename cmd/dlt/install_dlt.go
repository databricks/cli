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

func installDLTSymlink(directory string) error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}

	dir := directory
	if dir == "" {
		dir = filepath.Dir(path)
	}
	dltPath := filepath.Join(dir, "dlt")

	_, err = os.Lstat(dltPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		err = os.Symlink(realPath, dltPath)
		if err != nil {
			return err
		}
		cmdio.LogString(context.Background(), fmt.Sprintf("dlt successfully installed in directory %q", dir))
		return nil
	}

	target, err := filepath.EvalSymlinks(dltPath)
	if err != nil {
		return err
	}
	if realPath == target {
		cmdio.LogString(context.Background(), fmt.Sprintf("dlt already installed in directory %q", dir))
		return nil
	}
	return fmt.Errorf("cannot install dlt CLI: %q already exists", dltPath)
}

func InstallDLT() *cobra.Command {
	var directory string
	cmd := &cobra.Command{
		Use:    "install-dlt",
		Short:  "Install DLT",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installDLTSymlink(directory)
		},
	}
	cmd.Flags().StringVarP(&directory, "directory", "d", "", "Directory in which to install dlt CLI (defaults to databricks CLI's directory)")
	return cmd
}
