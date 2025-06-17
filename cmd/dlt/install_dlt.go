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
		return "", errors.New("databricks CLI executable not found")
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink: %w", err)
	}

	dir := directory
	if dir == "" {
		dir = filepath.Dir(path)
	}
	dltPath := filepath.Join(dir, "dlt")

	fi, err := os.Lstat(dltPath)
	if err == nil && fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(dltPath)
		if err == nil && target == realPath {
			cmdio.LogString(context.Background(), fmt.Sprintf("dlt already installed in directory %q", dir))
			return dltPath, nil
		}
		if err == nil && target != realPath {
			return "", fmt.Errorf("cannot install dlt CLI: %q already exists", dltPath)
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check if %q exists: %w", dltPath, err)
	}
	// Install the symlink.
	err = os.Symlink(realPath, dltPath)
	if err != nil {
		return "", fmt.Errorf("failed to install dlt CLI: %w", err)
	}
	cmdio.LogString(context.Background(), fmt.Sprintf("dlt successfully installed in directory %q", dir))
	return dltPath, nil
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
				// In text mode, just return success (the message is already logged by installDLTSymlink)
				return nil
			}
		},
	}
	cmd.Flags().StringVarP(&directory, "directory", "d", "", "Directory in which to install dlt CLI (defaults to databricks CLI's directory)")
	return cmd
}
