package debug

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/spf13/cobra"
)

// NewStatesCommand returns the `ucm debug states` command. Lists the state
// files ucm maintains for the selected target so users (and the VSCode
// extension) can see which engine's state the target is currently bound to.
func NewStatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "states",
		Short: "Show available state files",
		Args:  root.NoArgs,
	}

	// forcePull mirrors cmd/bundle/debug/states.go's flag so tooling that
	// invokes either subcommand has the same flag surface. TODO(#57): wire
	// this to ucm.deploy.Pull once the mirror path lands; for now it is a
	// no-op — ucm's pull/push path is still under construction.
	var forcePull bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u := utils.ProcessUcm(cmd, utils.ProcessOptions{})
		ctx := cmd.Context()
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		localDir := deploy.LocalStateDir(u)
		paths := []string{
			filepath.Join(localDir, deploy.UcmStateFileName),
			filepath.Join(localDir, "terraform", deploy.TfStateFileName),
			filepath.Join(localDir, direct.StateFileName),
		}

		var lines []string
		for _, p := range paths {
			line, ok, err := describePath(p)
			if err != nil {
				return err
			}
			if ok {
				lines = append(lines, line)
			}
		}

		out := cmd.OutOrStdout()
		if len(lines) == 0 {
			fmt.Fprintln(out, "No state files found.")
			return nil
		}
		fmt.Fprintln(out, strings.Join(lines, "\n"))
		return nil
	}

	return cmd
}

// describePath returns "<absolute-forward-slashed-path> (<size> bytes)" when
// the path exists, or (_, false, nil) when the file is absent. Paths are
// forward-slashed so acceptance-test output is stable across OSes (matches
// .agent/rules/style-guide-go.md).
func describePath(path string) (string, bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false, fmt.Errorf("abs %s: %w", filepath.ToSlash(path), err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("stat %s: %w", filepath.ToSlash(abs), err)
	}
	return fmt.Sprintf("%s (%d bytes)", filepath.ToSlash(abs), info.Size()), true, nil
}
