package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/auth"
	"github.com/databricks/cli/cmd/lakebox"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/databricks-sdk-go"
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
  lakebox auth login --host https://...   # authenticate to Databricks workspace and lakebox service
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

	authCmd := auth.New()
	// Hook into 'auth login' to auto-register SSH key after OAuth completes.
	for _, sub := range authCmd.Commands() {
		if sub.Name() == "login" {
			origRunE := sub.RunE
			sub.RunE = func(cmd *cobra.Command, args []string) error {
				// Run the original auth login.
				if err := origRunE(cmd, args); err != nil {
					return err
				}

				// Auto-register: generate lakebox SSH key and register it.
				fmt.Fprintln(cmd.ErrOrStderr(), "")
				fmt.Fprintln(cmd.ErrOrStderr(), "Setting up SSH access...")

				keyPath, pubKey, err := lakebox.EnsureAndReadKey()
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(),
						"SSH key setup failed: %v\n"+
							"You can set it up later with: lakebox register\n", err)
					return nil
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "Using SSH key: %s\n", keyPath)

				host := cmd.Flag("host").Value.String()
				if host == "" && len(args) > 0 {
					host = args[0]
				}
				profile := cmd.Flag("profile").Value.String()
				if profile == "" && host != "" {
					// Derive profile name the same way auth login does.
					h := strings.TrimPrefix(host, "https://")
					h = strings.TrimPrefix(h, "http://")
					profile = strings.SplitN(h, ".", 2)[0]
				}
				if profile != "" {
					if err := lakebox.SetLastProfile(profile); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to save last profile: %v\n", err)
					}
				}
				w, err := databricks.NewWorkspaceClient(&databricks.Config{
					Host:    host,
					Profile: profile,
				})
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(),
						"Could not initialize workspace client for key registration: %v\n"+
							"Run 'lakebox register' to complete setup.\n", err)
					return nil
				}

				if err := lakebox.RegisterKey(cmd.Context(), w, pubKey); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(),
						"Key registration failed: %v\n"+
							"Run 'lakebox register' to retry.\n", err)
					return nil
				}

				fmt.Fprintln(cmd.ErrOrStderr(), "SSH key registered. You're ready to use 'lakebox ssh'.")
				return nil
			}
			break
		}
	}
	cli.AddCommand(authCmd)

	// Register lakebox subcommands directly at root level.
	for _, sub := range lakebox.New().Commands() {
		cli.AddCommand(sub)
	}

	return cli
}
