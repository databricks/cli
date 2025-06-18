package dlt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

type installDLTResponse struct {
	SymlinkPath string `json:"symlink_path"`
}

func installDLTSymlink(directory string) (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}

	dir := directory
	if dir == "" {
		dir = filepath.Dir(path)
	}
	dltPath := filepath.Join(dir, "dlt")

	_, err = os.Lstat(dltPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		err = os.Symlink(realPath, dltPath)
		if err != nil {
			return "", err
		}
		cmdio.LogString(context.Background(), fmt.Sprintf("dlt successfully installed in directory %q", dir))
		return dltPath, nil
	}

	target, err := filepath.EvalSymlinks(dltPath)
	if err == nil && realPath == target {
		cmdio.LogString(context.Background(), fmt.Sprintf("dlt already installed in directory %q", dir))
		return dltPath, nil
	}
	cmdio.LogString(context.Background(), fmt.Sprintf("cannot install dlt CLI: %q already exists", dltPath))
	if err != nil {
		return "", err
	}
	return "", errors.New("installation failed")
}

func InstallDLT() *cobra.Command {
	var directory string
	cmd := &cobra.Command{
		Use:    "install-dlt",
		Short:  "Install DLT",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dltPath, err := installDLTSymlink(directory)
			if err != nil {
				return err
			}
			response := installDLTResponse{
				SymlinkPath: dltPath,
			}
			switch root.OutputType(cmd) {
			case flags.OutputJSON:
				return cmdio.Render(cmd.Context(), response)
			default:
				return nil
			}
		},
	}
	cmd.Flags().StringVarP(&directory, "directory", "d", "", "Directory in which to install dlt CLI (defaults to databricks CLI's directory)")
	return cmd
}
