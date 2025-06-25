package pipelines

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func installPipelinesSymlink(ctx context.Context, directory string) error {
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
	pipelinesPath := filepath.Join(dir, "pipelines")

	_, err = os.Lstat(pipelinesPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		err = os.Symlink(realPath, pipelinesPath)
		if err != nil {
			return err
		}
		cmdio.LogString(ctx, fmt.Sprintf("pipelines successfully installed in directory %q", dir))
		return nil
	}

	target, err := filepath.EvalSymlinks(pipelinesPath)
	if err != nil {
		return err
	}
	if realPath == target {
		cmdio.LogString(ctx, fmt.Sprintf("pipelines already installed in directory %q", dir))
		return nil
	}
	return fmt.Errorf("cannot install pipelines CLI: %q already exists", pipelinesPath)
}

func InstallPipelinesCLI() *cobra.Command {
	var directory string
	cmd := &cobra.Command{
		Use:    "install-pipelines-cli",
		Short:  "Install Pipelines CLI",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installPipelinesSymlink(cmd.Context(), directory)
		},
	}
	cmd.Flags().StringVarP(&directory, "directory", "d", "", "Directory in which to install pipelines CLI (defaults to databricks CLI's directory)")
	return cmd
}
