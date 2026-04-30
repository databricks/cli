package postgrescmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

// defaultDatabase is the database name used when --database is not set.
// Lakebase Autoscaling and Provisioned both use this name as their default.
const defaultDatabase = "databricks_postgres"

// queryFlags is the union of every flag the query command exposes. Lifted
// out of newQueryCmd so unit-tested helpers (resolveTarget, etc.) can take
// it directly without poking at cobra internals.
type queryFlags struct {
	targetingFlags
	database       string
	connectTimeout time.Duration
	maxRetries     int

	// outputFormat is the raw flag value. resolveOutputFormat turns it into
	// the effective format (which may differ when stdout is piped).
	outputFormat    string
	outputFormatSet bool
}

func newQueryCmd() *cobra.Command {
	var f queryFlags

	cmd := &cobra.Command{
		Use:   "query [SQL]",
		Short: "Run a SQL statement against a Lakebase Postgres endpoint",
		Long: `Execute a single SQL statement against a Lakebase Postgres endpoint.

Targeting (exactly one form required):
  --target STRING       Provisioned instance name OR autoscaling resource path
                        (e.g. my-instance, projects/foo/branches/main/endpoints/primary)
  --project ID          Autoscaling project ID
  --branch ID           Autoscaling branch ID (default: auto-select if exactly one)
  --endpoint ID         Autoscaling endpoint ID

Output:
  --output text         Aligned table for rows-producing statements (default).
                        Falls back to JSON when stdout is not a terminal so
                        scripts piping the output get machine-readable results.
  --output json         Top-level array of row objects, streamed for
                        rows-producing statements. Command-only statements
                        emit a single {"command": "...", "rows_affected": N}
                        object. Numbers, booleans, NULL, jsonb, timestamps
                        render with their JSON-native types.
  --output csv          Header row + one CSV row per result row, streamed.
                        Command-only statements write the command tag to
                        stderr.

DATABRICKS_OUTPUT_FORMAT is honoured when --output is not explicitly set.

This is an experimental command. The flag set, output shape, and supported
target kinds will expand in subsequent releases.

Limitations (this release):

  - Single SQL statement per invocation (multi-statement support comes later).
  - No interactive REPL. 'databricks psql' continues to own that surface.
  - Multi-statement strings (e.g. "SELECT 1; SELECT 2") are not supported.
  - The OAuth token is generated once per invocation and is valid for 1h.
    Queries longer than that fail with an auth error.
`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			f.outputFormatSet = cmd.Flag("output").Changed
			return runQuery(cmd.Context(), cmd, args[0], f)
		},
	}

	cmd.Flags().StringVar(&f.target, "target", "", "Provisioned instance name OR autoscaling resource path")
	cmd.Flags().StringVar(&f.project, "project", "", "Autoscaling project ID")
	cmd.Flags().StringVar(&f.branch, "branch", "", "Autoscaling branch ID (default: auto-select if exactly one)")
	cmd.Flags().StringVar(&f.endpoint, "endpoint", "", "Autoscaling endpoint ID (default: auto-select if exactly one)")
	cmd.Flags().StringVarP(&f.database, "database", "d", defaultDatabase, "Database name")
	cmd.Flags().DurationVar(&f.connectTimeout, "connect-timeout", defaultConnectTimeout, "Connect timeout")
	cmd.Flags().IntVar(&f.maxRetries, "max-retries", 3, "Total connect attempts on idle/waking endpoint (must be >= 1; 1 disables retry)")
	cmd.Flags().StringVarP(&f.outputFormat, "output", "o", string(outputText), "Output format: text, json, or csv")
	cmd.RegisterFlagCompletionFunc("output", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		out := make([]string, len(allOutputFormats))
		for i, f := range allOutputFormats {
			out[i] = string(f)
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.MarkFlagsMutuallyExclusive("target", "project")
	cmd.MarkFlagsMutuallyExclusive("target", "branch")
	cmd.MarkFlagsMutuallyExclusive("target", "endpoint")

	return cmd
}

// runQuery is the production entry point. It is split out from RunE so unit
// tests can call it directly with a stubbed connectFunc once we add seam-based
// tests in a later PR.
func runQuery(ctx context.Context, cmd *cobra.Command, sql string, f queryFlags) error {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return errors.New("no SQL provided")
	}
	if f.maxRetries < 1 {
		return fmt.Errorf("--max-retries must be at least 1; got %d", f.maxRetries)
	}
	if err := validateTargeting(f.targetingFlags); err != nil {
		return err
	}

	// SupportsColor is the public TTY-ish signal libs/cmdio exposes today; it
	// also folds in NO_COLOR / TERM=dumb, which strictly speaking are colour
	// preferences rather than TTY signals. Users who hit that edge case can
	// pass --output text explicitly; that path is honoured (see
	// resolveOutputFormat). Mirrors the aitools query command.
	stdoutTTY := cmdio.SupportsColor(ctx, cmd.OutOrStdout())
	format, err := resolveOutputFormat(ctx, f.outputFormat, f.outputFormatSet, stdoutTTY)
	if err != nil {
		return err
	}

	resolved, err := resolveTarget(ctx, f.targetingFlags)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, fmt.Sprintf("Connecting to %s...", resolved.DisplayName))

	pgxCfg, err := buildPgxConfig(connectConfig{
		Host:           resolved.Host,
		Port:           5432,
		Username:       resolved.Username,
		Password:       resolved.Token,
		Database:       f.database,
		ConnectTimeout: f.connectTimeout,
	})
	if err != nil {
		return err
	}

	rc := retryConfig{
		MaxAttempts:  f.maxRetries,
		InitialDelay: time.Second,
		MaxDelay:     10 * time.Second,
	}

	conn, err := connectWithRetry(ctx, pgxCfg, rc, pgx.ConnectConfig)
	if err != nil {
		return err
	}
	defer conn.Close(context.WithoutCancel(ctx))

	sink := newSink(format, cmd.OutOrStdout(), cmd.ErrOrStderr())
	return executeOne(ctx, conn, sql, sink)
}

// newSink returns the rowSink for the chosen output format. Kept separate
// from runQuery so tests can build sinks without going through pgx.
func newSink(format outputFormat, out, stderr io.Writer) rowSink {
	switch format {
	case outputJSON:
		return newJSONSink(out, stderr)
	case outputCSV:
		return newCSVSink(out, stderr)
	default:
		return newTextSink(out)
	}
}
