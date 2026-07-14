package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newSSHKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage SSH keys registered with the sandbox service",
	}
	cmd.AddCommand(newSSHKeyListCommand())
	cmd.AddCommand(newSSHKeyDeleteCommand())
	return cmd
}

func newSSHKeyDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <key-hash>",
		Short: "Delete an SSH key registered with the sandbox service",
		Long: `Delete an SSH key registered with the sandbox service.

The key hash is the identifier shown by 'databricks sandbox ssh-key list'.
Once deleted, future SSH attempts authenticated by the corresponding
private key will fail until the key is re-registered.

Example:
  databricks sandbox ssh-key delete a1b2c3d4e5f6...`,
		Args:              cobra.ExactArgs(1),
		PreRunE:           root.MustWorkspaceClient,
		ValidArgsFunction: completeSSHKeyHashes,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newSandboxAPI(w)
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

func newSSHKeyListCommand() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List SSH keys registered with the sandbox service",
		Long: `List SSH keys registered with the sandbox service.

Each row shows the server-assigned key hash (the identifier used to
delete the key), the user-supplied name, and create / last-use
timestamps. The locally-registered key (from 'databricks sandbox
register') is marked when its hash matches one of the listed entries.

Examples:
  databricks sandbox ssh-key list
  databricks sandbox ssh-key list --json`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newSandboxAPI(w)
			if err != nil {
				return err
			}

			keys, err := api.listKeys(ctx)
			if err != nil {
				return fmt.Errorf("failed to list ssh keys: %w", err)
			}

			if jsonOutput(cmd, outputJSON) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(keys)
			}

			if len(keys) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n",
					cmdio.Faint(ctx, "No SSH keys registered. Run 'databricks sandbox register' to add one."))
				return nil
			}

			// Best-effort: compute the hash of the locally-registered key so
			// we can highlight which row belongs to this machine. Missing key
			// file or read errors are non-fatal — just skip the marker.
			localHash := ""
			if path, err := sandboxKeyPath(ctx); err == nil {
				if data, err := os.ReadFile(path + ".pub"); err == nil {
					localHash = keyHash(string(data))
				}
			}

			out := cmd.OutOrStdout()
			blank(out)

			// Measure in terminal cells (cmdio.Width) so wide / emoji
			// glyphs in `--name` don't misalign the row.
			nameCol := 4
			for _, k := range keys {
				if l := cmdio.Width(k.Name); l > nameCol {
					nameCol = l
				}
			}
			nameCol += 2
			const hashCol = 32
			const timeCol = 20

			// 4-char gutter holds a per-row `*` for the local key (blank in header).
			header := fmt.Sprintf("%-*s  %-*s  %-*s  %s",
				nameCol, "NAME", hashCol, "KEY HASH", timeCol, "CREATED", "LAST USED")
			fmt.Fprintf(out, "    %s\n", cmdio.Faint(ctx, header))
			fmt.Fprintf(out, "    %s\n", cmdio.Faint(ctx, strings.Repeat("─", nameCol+hashCol+timeCol+timeCol+6)))

			localFound := false
			for _, k := range keys {
				// cmdio.PadRight measures visible width, so the ANSI escapes
				// cmdio.Faint wraps the NAME cell in don't break alignment.
				displayName := k.Name
				if displayName == "" {
					displayName = cmdio.Faint(ctx, "(unset)")
				}
				gutter := "    "
				if localHash != "" && k.KeyHash == localHash {
					gutter = "  " + cmdio.Cyan(ctx, "*") + " "
					localFound = true
				}
				fmt.Fprintf(out, "%s%s  %-*s  %-*s  %s\n",
					gutter,
					cmdio.PadRight(displayName, nameCol),
					hashCol, k.KeyHash,
					timeCol, formatTimeShort(k.CreateTime),
					formatTimeShort(k.LastUseTime))
			}
			// Without a legend the `*` (or its absence) is opaque. Print the
			// meaning either way so users can tell "no `*` on any row" apart
			// from "this terminal doesn't print the marker".
			blank(out)
			switch {
			case localFound:
				fmt.Fprintf(out, "  %s\n", cmdio.Faint(ctx, cmdio.Cyan(ctx, "*")+" key matches the one on this machine"))
			case localHash != "":
				fmt.Fprintf(out, "  %s\n", cmdio.Faint(ctx, "(no registered key matches this machine's local key — run `databricks sandbox register` to register it)"))
			default:
				fmt.Fprintf(out, "  %s\n", cmdio.Faint(ctx, "(no local sandbox key on this machine — run `databricks sandbox register` to create and register one)"))
			}
			blank(out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")
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
