package cmd

import (
	"context"

	"github.com/databricks/cli/cmd/auth"
	"github.com/databricks/cli/cmd/lakebox"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	cli := root.New(ctx)
	cli.Use = "lakebox"
	cli.Short = "Lakebox CLI — manage Databricks sandbox environments"
	cli.Long = `Lakebox CLI — manage Databricks sandbox environments.

Lakebox provides SSH-accessible development environments backed by
microVM isolation. Each lakebox is a personal sandbox with pre-installed
tooling (Python, Node.js, Rust, Databricks CLI) and persistent storage.

Getting started:
  lakebox auth login --host https://...   # authenticate to Databricks
  lakebox register                        # generate SSH key and register
  lakebox ssh                             # SSH to your default lakebox

Common workflows:
  lakebox ssh                             # SSH to your default lakebox
  lakebox ssh my-project                  # SSH to a named lakebox
  lakebox list                            # list your lakeboxes
  lakebox create                          # create a new lakebox
  lakebox delete my-project               # delete a lakebox
  lakebox status my-project               # show lakebox status

The CLI manages your ~/.ssh/config so you can also connect directly:
  ssh my-project                          # after 'lakebox ssh'
`
	cli.CompletionOptions.DisableDefaultCmd = true

	cli.AddCommand(auth.New())

	// Register lakebox subcommands directly at root level.
	for _, sub := range lakebox.New().Commands() {
		cli.AddCommand(sub)
	}

	return cli
}
