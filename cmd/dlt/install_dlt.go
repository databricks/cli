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

	_, err = os.Lstat(dltPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("failed to check if %q exists: %w", dltPath, err)
		}
		err = os.Symlink(realPath, dltPath)
		if err != nil {
			return "", fmt.Errorf("failed to install dlt CLI: %w", err)
		}
		cmdio.LogString(context.Background(), fmt.Sprintf("dlt successfully installed in directory %q", dir))
		return dltPath, nil
	}

	target, readErr := filepath.EvalSymlinks(dltPath)
	if readErr != nil {
		return "", fmt.Errorf("cannot install dlt CLI: %q already exists", dltPath)
	}
	normalizedTarget := filepath.ToSlash(filepath.Clean(target))
	normalizedRealPath := filepath.ToSlash(filepath.Clean(realPath))
	if normalizedTarget == normalizedRealPath {
		cmdio.LogString(context.Background(), fmt.Sprintf("dlt already installed in directory %q", dir))
		return dltPath, nil
	}
	return "", fmt.Errorf("cannot install dlt CLI: %q already exists", dltPath)
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
