package gitcredentials

import (
	git_credentials "github.com/databricks/bricks/cmd/gitcredentials/git-credentials"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "gitcredentials",
}

func init() {

	Cmd.AddCommand(git_credentials.Cmd)
}
