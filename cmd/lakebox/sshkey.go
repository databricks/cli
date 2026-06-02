package lakebox

import (
	"fmt"
	"os"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func newSSHKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage SSH keys registered with the lakebox service",
	}
	cmd.AddCommand(newSSHKeyListCommand())
	cmd.AddCommand(newSSHKeyDeleteCommand())
	return cmd
}

func newSSHKeyDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <key-hash>",
		Short: "Delete an SSH key registered with the lakebox service",
		Long: `Delete an SSH key registered with the lakebox service.

The key hash is the identifier shown by 'databricks lakebox ssh-key list'.
Once deleted, future SSH attempts authenticated by the corresponding
private key will fail until the key is re-registered.

Example:
  databricks lakebox ssh-key delete a1b2c3d4e5f6...`,
		Args:              cobra.ExactArgs(1),
		PreRunE:           root.MustWorkspaceClient,
		ValidArgsFunction: completeSSHKeyHashes,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			hash := args[0]
			s := spin(ctx, "Deleting key "+hash+"…")
			defer s.Close()
			if err := api.deleteKey(ctx, hash); err != nil {
				s.fail("Failed to delete key")
				return fmt.Errorf("failed to delete ssh key %s: %w", hash, err)
			}
			s.ok("SSH key " + cmdio.Bold(ctx, hash) + " deleted")
			return nil
		},
	}
	return cmd
}

// sshKeyRow embeds sshKeyEntry so JSON output stays byte-identical to
// the raw API response. Text-mode fields are tagged `json:"-"` so they
// don't leak when the user passes `-o json`.
type sshKeyRow struct {
	sshKeyEntry
	DisplayName string `json:"-"`
	Created     string `json:"-"`
	LastUsed    string `json:"-"`
	Local       string `json:"-"`
}

const (
	sshKeyHeaderTemplate = `{{header "LOCAL"}}	{{header "NAME"}}	{{header "KEY HASH"}}	{{header "CREATED"}}	{{header "LAST USED"}}`
	sshKeyRowTemplate    = `{{range .}}{{.Local | cyan}}	{{.DisplayName}}	{{.KeyHash}}	{{.Created | faint}}	{{.LastUsed | faint}}
{{end}}`
)

func newSSHKeyListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List SSH keys registered with the lakebox service",
		Long: `List SSH keys registered with the lakebox service.

Each row shows the server-assigned key hash (the identifier used to
delete the key), the user-supplied name, and create / last-use
timestamps. The locally-registered key (from 'databricks lakebox
register') is marked with a '*' in the LOCAL column.

Examples:
  databricks lakebox ssh-key list
  databricks lakebox ssh-key list -o json`,
		PreRunE: root.MustWorkspaceClient,
		Annotations: map[string]string{
			"headerTemplate": sshKeyHeaderTemplate,
			"template":       sshKeyRowTemplate,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			keys, err := api.listKeys(ctx)
			if err != nil {
				return fmt.Errorf("failed to list ssh keys: %w", err)
			}

			if root.OutputType(cmd) == flags.OutputJSON {
				return cmdio.Render(ctx, keys)
			}

			if len(keys) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n",
					cmdio.Faint(ctx, "No SSH keys registered. Run 'databricks lakebox register' to add one."))
				return nil
			}

			// Best-effort: compute the hash of the locally-registered key
			// so we can highlight which row belongs to this machine.
			// Missing key file or read errors are non-fatal — just skip
			// the marker.
			localHash := ""
			if path, err := lakeboxKeyPath(ctx); err == nil {
				if data, err := os.ReadFile(path + ".pub"); err == nil {
					localHash = keyHash(string(data))
				}
			}

			rows := make([]sshKeyRow, len(keys))
			localFound := false
			for i, k := range keys {
				name := k.Name
				if name == "" {
					name = "-"
				}
				local := ""
				if localHash != "" && k.KeyHash == localHash {
					local = "*"
					localFound = true
				}
				rows[i] = sshKeyRow{
					sshKeyEntry: k,
					DisplayName: name,
					Created:     formatTimeShort(k.CreateTime),
					LastUsed:    formatTimeShort(k.LastUseTime),
					Local:       local,
				}
			}

			if err := cmdio.Render(ctx, rows); err != nil {
				return err
			}

			// Without a legend the `*` (or its absence) is opaque. Print
			// the meaning either way so users can tell "no `*` on any
			// row" apart from "this terminal doesn't print the marker".
			switch {
			case localFound:
				cmdio.LogString(ctx, "  "+cmdio.Faint(ctx, cmdio.Cyan(ctx, "*")+" key matches the one on this machine"))
			case localHash != "":
				cmdio.LogString(ctx, "  "+cmdio.Faint(ctx, "(no registered key matches this machine's local key — run `databricks lakebox register` to register it)"))
			default:
				cmdio.LogString(ctx, "  "+cmdio.Faint(ctx, "(no local lakebox key on this machine — run `databricks lakebox register` to create and register one)"))
			}
			return nil
		},
	}
	return cmd
}

// formatTimeShort renders an RFC 3339 timestamp as a short, compact date
// for table display. Returns "-" for empty input; passes through the raw
// value if it doesn't parse (so server-side schema changes don't silently
// hide data).
func formatTimeShort(rfc3339 string) string {
	if rfc3339 == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	return t.Format("2006-01-02 15:04")
}
