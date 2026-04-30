package postgrescmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/libs/sqlcli"
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
	files          []string

	// outputFormat is the raw flag value. resolveOutputFormat turns it into
	// the effective format (which may differ when stdout is piped).
	outputFormat    string
	outputFormatSet bool
}

func newQueryCmd() *cobra.Command {
	var f queryFlags

	cmd := &cobra.Command{
		Use:   "query [SQL | file.sql]...",
		Short: "Run SQL statements against a Lakebase Postgres endpoint",
		Long: `Execute one or more SQL statements against a Lakebase Postgres endpoint.

Targeting (exactly one form required):
  --target STRING       Provisioned instance name OR autoscaling resource path
                        (e.g. my-instance, projects/foo/branches/main/endpoints/primary)
  --project ID          Autoscaling project ID
  --branch ID           Autoscaling branch ID (default: auto-select if exactly one)
  --endpoint ID         Autoscaling endpoint ID

Inputs (positionals and --file may be combined; execution order is files-first
then positionals; stdin is used only when neither is present):
  -f, --file PATH       SQL file path (repeatable). Each file must contain
                        exactly one statement.
  positional            SQL string OR path ending in '.sql' that exists on disk.

Output:
  --output text         Aligned table for rows-producing statements (default).
                        Falls back to JSON when stdout is not a terminal so
                        scripts piping the output get machine-readable results.
  --output json         For a single input: top-level array of row objects,
                        streamed. For multiple inputs: top-level array of
                        per-unit result objects ({"sql","kind","elapsed_ms",...}),
                        with each object buffered to completion.
  --output csv          Header row + one CSV row per result row, streamed.
                        Single-input only; multi-input + csv is rejected
                        pre-flight. Use --output json for multi-input.

DATABRICKS_OUTPUT_FORMAT is honoured when --output is not explicitly set.

Limitations (this release):

  - Single statement per input unit. Multi-statement strings (e.g.
    "SELECT 1; SELECT 2") are rejected; pass each as a separate positional
    or --file.
  - No interactive REPL. 'databricks psql' continues to own that surface.
  - Inputs run sequentially on one connection; session state (SET, temp
    tables, prepared statement names) carries across them.
  - The OAuth token is generated once per invocation and is valid for 1h.
    Queries longer than that fail with an auth error.
  - --output csv is rejected when more than one input unit is present;
    use --output json or split into separate invocations.
`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			f.outputFormatSet = cmd.Flag("output").Changed
			return runQuery(cmd.Context(), cmd, args, f)
		},
	}

	cmd.Flags().StringVar(&f.target, "target", "", "Provisioned instance name OR autoscaling resource path")
	cmd.Flags().StringVar(&f.project, "project", "", "Autoscaling project ID")
	cmd.Flags().StringVar(&f.branch, "branch", "", "Autoscaling branch ID (default: auto-select if exactly one)")
	cmd.Flags().StringVar(&f.endpoint, "endpoint", "", "Autoscaling endpoint ID (default: auto-select if exactly one)")
	cmd.Flags().StringVarP(&f.database, "database", "d", defaultDatabase, "Database name")
	cmd.Flags().DurationVar(&f.connectTimeout, "connect-timeout", defaultConnectTimeout, "Connect timeout")
	cmd.Flags().IntVar(&f.maxRetries, "max-retries", 3, "Total connect attempts on idle/waking endpoint (must be >= 1; 1 disables retry)")
	cmd.Flags().StringArrayVarP(&f.files, "file", "f", nil, "SQL file path (repeatable)")
	cmd.Flags().StringVarP(&f.outputFormat, "output", "o", string(sqlcli.OutputText), "Output format: text, json, or csv")
	cmd.RegisterFlagCompletionFunc("output", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		out := make([]string, len(sqlcli.AllFormats))
		for i, f := range sqlcli.AllFormats {
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
func runQuery(ctx context.Context, cmd *cobra.Command, args []string, f queryFlags) error {
	if f.maxRetries < 1 {
		return fmt.Errorf("--max-retries must be at least 1; got %d", f.maxRetries)
	}
	if err := validateTargeting(f.targetingFlags); err != nil {
		return err
	}

	units, err := sqlcli.Collect(ctx, cmd.InOrStdin(), args, f.files, sqlcli.CollectOptions{})
	if err != nil {
		return err
	}
	for _, u := range units {
		if err := checkSingleStatement(u.SQL); err != nil {
			return fmt.Errorf("%s: %w%s", u.Source, err, multiStatementHint)
		}
	}

	stdoutTTY := cmdio.SupportsColor(ctx, cmd.OutOrStdout())
	format, err := sqlcli.ResolveFormat(ctx, f.outputFormat, f.outputFormatSet, stdoutTTY)
	if err != nil {
		return err
	}

	// CSV multi-input is rejected pre-flight: there is no sensible shape for
	// a CSV that has to merge schemas across statements. The error names the
	// flag pair and tells the user how to recover, per the repo rule about
	// rejecting incompatible inputs early.
	if format == sqlcli.OutputCSV && len(units) > 1 {
		return fmt.Errorf("--output csv requires a single input unit; got %d (use --output json for multi-input invocations)", len(units))
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

	out := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

	if len(units) == 1 {
		// Single-input path: stream directly through the per-format sink.
		// Avoids buffering rows for large exports and matches the v1 single-
		// input behaviour PR 2 shipped. Wrap the error so DETAIL / HINT
		// from a *pgconn.PgError surface even on the single-input path.
		sink := newSink(format, out, stderr)
		if err := executeOne(ctx, conn, units[0].SQL, sink); err != nil {
			return errors.New(formatPgError(err))
		}
		return nil
	}

	// Multi-input path: per-unit buffering. The plan accepts this trade-off
	// (multi-input invocations with huge SELECTs should use single-input
	// invocations with --output csv for streaming). Session state (SET,
	// temp tables) carries across units because we hold the same connection.
	results := make([]*unitResult, 0, len(units))
	for _, u := range units {
		r, err := runUnitBuffered(ctx, conn, u)
		if err != nil {
			// Render the successful prefix, then surface the error with
			// rich pgError formatting if applicable.
			if rerr := renderPartial(out, stderr, format, results, r, err); rerr != nil {
				// Best-effort partial render failed; surface the original
				// error to the user, the renderer error to debug logs.
				fmt.Fprintln(stderr, "warning: failed to render partial result:", rerr)
			}
			return formatExecutionError(u.Source, err)
		}
		results = append(results, r)
	}

	switch format {
	case sqlcli.OutputJSON:
		return renderJSONMulti(out, stderr, results, nil, "")
	default:
		return renderTextMulti(out, results)
	}
}

// newSink returns the rowSink for the chosen output format. Kept separate
// from runQuery so tests can build sinks without going through pgx.
func newSink(format sqlcli.Format, out, stderr io.Writer) rowSink {
	switch format {
	case sqlcli.OutputJSON:
		return newJSONSink(out, stderr)
	case sqlcli.OutputCSV:
		return newCSVSink(out, stderr)
	default:
		return newTextSink(out)
	}
}

// renderPartial emits the rendered output for the prefix of units that ran
// successfully before a unit errored. For multi-input json this also writes
// the error envelope as the last array element.
func renderPartial(out, stderr io.Writer, format sqlcli.Format, results []*unitResult, errored *unitResult, err error) error {
	switch format {
	case sqlcli.OutputJSON:
		return renderJSONMulti(out, stderr, results, errored, formatPgError(err))
	default:
		// Text: render whatever ran cleanly. The error message goes through
		// cobra's default error path on stderr after we return.
		return renderTextMulti(out, results)
	}
}

// formatExecutionError produces the error returned to cobra when an input
// unit failed. The message includes the source label so the user knows
// which of N inputs blew up.
func formatExecutionError(source string, err error) error {
	return fmt.Errorf("%s: %s", source, formatPgError(err))
}

// multiStatementHint is appended to errMultipleStatements so users see the
// recovery path inline.
const multiStatementHint = "\nThis command runs one statement per input. To run multiple statements:\n" +
	`  - Pass each as a separate positional:  query "SELECT 1" "SELECT 2"` + "\n" +
	`  - Pass each in its own --file:         query --file q1.sql --file q2.sql`
