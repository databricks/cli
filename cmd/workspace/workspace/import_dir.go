package workspace

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sync"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// TODO: check whether we need mutex for any events been emitted since they are accessing
// state
// TODO: Error: path must be nested under /Users/shreyas.goenka@databricks.com or /Repos/shreyas.goenka@databricks.com. Should this validation be
// removed? Yes
var importDirCmd = &cobra.Command{
	Use:   "import-dir SOURCE_PATH TARGET_PATH",
	Short: `Import directory to a Databricks workspace.`,
	Long: `
  Recursively imports a directory from  the local filesystem to a Databricks workspace.

  This command respects your git ignore configuration. Notebooks with extensions
  .scala, .py, .sql, .r, .R, .ipynb are stripped of their extensions.
`,

	PreRunE: root.MustWorkspaceClient,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		sourcePath := args[0]
		targetPath := args[1]

		// Initialize syncer to do a full sync with the correct from source to target.
		// This will upload the local files
		opts := sync.SyncOptions{
			LocalPath:       sourcePath,
			RemotePath:      targetPath,
			Full:            true,
			WorkspaceClient: root.WorkspaceClient(ctx),

			AllowOverwrites: importDirOverwrite,
			PersistSnapshot: false,
		}
		s, err := sync.New(ctx, opts)
		if err != nil {
			return err
		}

		// Initialize error wait group, and spawn the progress event emitter inside
		// the error wait group
		group, ctx := errgroup.WithContext(ctx)
		eventsChannel := s.Events()
		group.Go(
			func() error {
				return renderSyncEvents(ctx, eventsChannel, s)
			})

		// Start Uploading local files
		cmdio.RenderWithTemplate(ctx, newImportStartedEvent(sourcePath, targetPath), `Starting import {{.SourcePath}} -> {{TargetPath}}`)
		err = s.RunOnce(ctx)
		if err != nil {
			return err
		}
		// Upload completed, close the syncer
		s.Close()

		// Wait for any inflight progress events to be emitted
		if err := group.Wait(); err != nil {
			return err
		}

		// Render import completetion event
		cmdio.RenderWithTemplate(ctx, newImportCompleteEvent(sourcePath, targetPath), `Completed import. Files available at {{.TargetPath}}`)
		return nil
	},
}

var importDirOverwrite bool

func init() {
	importDirCmd.Flags().BoolVar(&importDirOverwrite, "overwrite", false, "Overwrite if file already exists in the workspace")
	Cmd.AddCommand(importDirCmd)
}
