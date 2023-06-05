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
var importDirCmd = &cobra.Command{
	Use:   "import-dir SOURCE_PATH TARGET_PATH",
	Short: `Recursively imports a directory from local to the Databricks workspace.`,
	Long: `
  Imports directory to the workspace.

  This command respects your git ignore configuration. Notebooks with extensions
  .scala, .py, .sql, .r, .R, .ipynb are stripped of their extensions.
`,

	Annotations: map[string]string{
		// TODO: use render with template at individual call sites for these events.
		"template": cmdio.Heredoc(`
		{{if eq .Type "IMPORT_STARTED"}}Import started
		{{else if eq .Type "UPLOAD_COMPLETE"}}Uploaded {{.SourcePath}} -> {{.TargetPath}}
		{{else if eq .Type "IMPORT_COMPLETE"}}Import completed
		{{end}}
		`),
	},
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
		cmdio.Render(ctx, newImportStartedEvent(sourcePath, targetPath))
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
		cmdio.Render(ctx, newImportCompleteEvent(sourcePath, targetPath))
		return nil
	},
}

var importDirOverwrite bool

func init() {
	importDirCmd.Flags().BoolVar(&importDirOverwrite, "overwrite", false, "Overwrite if file already exists in the workspace")
	Cmd.AddCommand(importDirCmd)
}
