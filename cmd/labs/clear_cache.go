package labs

import (
	"log/slog"
	"os"

	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func newClearCacheCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clear-cache",
		Short: "Clears cache entries from everywhere relevant",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			projects, err := project.Installed(ctx)
			if err != nil {
				return err
			}
			cache, err := project.PathInLabs(ctx, "databrickslabs-repositories.json")
			if err != nil {
				return err
			}
			_ = os.Remove(cache)
			logger := log.GetLogger(ctx)
			for _, prj := range projects {
				logger.Info("clearing labs project cache", slog.String("name", prj.Name))
				_ = os.RemoveAll(prj.CacheDir())
				// recreating empty cache folder for downstream apps to work normally
				_ = prj.EnsureFoldersExist()
			}
			return nil
		},
	}
}
