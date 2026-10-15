package shares

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	sharing "github.com/databricks/cli/libs/sdktypes/sharing/v1"
	"github.com/spf13/cobra"
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sharing.ListSharesRequest

	flags.IntPtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Maximum number of shares to return.`)
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list"
	cmd.Short = `List shares (Deprecated).`
	cmd.Long = `List shares (Deprecated).

  Gets an array of data object shares from the metastore. The caller must be a
  metastore admin or the owner of the share. There is no guarantee of a specific
  ordering of the elements in the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		sdkReq, err := sharing.ListSharesRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		response := w.Sharing.ListSharesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, sharing.ShareInfoFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	return cmd
}

func cmdOverride(cmd *cobra.Command) {
	// List command override is added here because the command is deprecated
	// and removed from the API definition. Use `list-shares` instead.
	cmd.AddCommand(newList())
}

func init() {
	cmdOverrides = append(cmdOverrides, cmdOverride)
}
