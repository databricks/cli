package postgrescmd

import (
	"context"
	"errors"
	"fmt"
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
}

func newQueryCmd() *cobra.Command {
	var f queryFlags

	cmd := &cobra.Command{
		Use:   "query [SQL]",
		Short: "Run a SQL statement against a Lakebase Postgres endpoint",
		Long: `Execute a single SQL statement against a Lakebase Postgres endpoint and
render the result as text.

Targeting (exactly one form required):
  --target STRING       Autoscaling resource path
                        (e.g. projects/foo/branches/main/endpoints/primary)
  --project ID          Autoscaling project ID
  --branch ID           Autoscaling branch ID (default: auto-select if exactly one)
  --endpoint ID         Autoscaling endpoint ID (default: auto-select if exactly one)

This is an experimental command. The flag set, output shape, and supported
target kinds will expand in subsequent releases.

Limitations (this release):

  - Single SQL statement per invocation (multi-statement support comes later).
  - Text output only. JSON and CSV output come in a follow-up release.
  - Only Lakebase Autoscaling endpoints are supported. Provisioned instance
    support comes in a follow-up release; use 'databricks psql <instance>' as a
    workaround for now.
  - No interactive REPL. 'databricks psql' continues to own that surface.
  - Multi-statement strings (e.g. "SELECT 1; SELECT 2") are not supported.
  - The OAuth token is generated once per invocation and is valid for 1h.
    Queries longer than that fail with an auth error.
`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuery(cmd.Context(), cmd, args[0], f)
		},
	}

	cmd.Flags().StringVar(&f.target, "target", "", "Autoscaling resource path (e.g. projects/foo/branches/main/endpoints/primary)")
	cmd.Flags().StringVar(&f.project, "project", "", "Autoscaling project ID")
	cmd.Flags().StringVar(&f.branch, "branch", "", "Autoscaling branch ID (default: auto-select if exactly one)")
	cmd.Flags().StringVar(&f.endpoint, "endpoint", "", "Autoscaling endpoint ID (default: auto-select if exactly one)")
	cmd.Flags().StringVarP(&f.database, "database", "d", defaultDatabase, "Database name")
	cmd.Flags().DurationVar(&f.connectTimeout, "connect-timeout", defaultConnectTimeout, "Connect timeout")
	cmd.Flags().IntVar(&f.maxRetries, "max-retries", 3, "Total connect attempts on idle/waking endpoint (must be >= 1; 1 disables retry)")

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

	result, err := executeOne(ctx, conn, sql)
	if err != nil {
		return err
	}

	return renderText(cmd.OutOrStdout(), result)
}
