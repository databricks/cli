package query

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/databricks/cli/cmd/query/internal/executor"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/safety/sqlsafe"
)

type sqlFormat = executor.Format

const (
	formatTable sqlFormat = executor.FormatTable
	formatJSON  sqlFormat = executor.FormatJSON
	formatCSV   sqlFormat = executor.FormatCSV
)

const envAllowDestructive = "DATABRICKS_CLI_ALLOW_DESTRUCTIVE_SQL"

var destructiveOverride = sqlsafe.OverrideHelper{
	FlagName:   "--allow-destructive",
	EnvVar:     envAllowDestructive,
	ProfileKey: "cli.allow_destructive_sql",
}

type sqlOptions struct {
	warehouseID          string
	inlineSQL            string
	file                 string
	rawFormat            string
	waitTimeout          time.Duration
	resultFile           string
	allowDestructiveFlag bool
}

func newSQLCommand() *cobra.Command {
	opts := &sqlOptions{}

	cmd := &cobra.Command{
		Use:   "sql",
		Short: "Execute SQL using the Databricks Statement Execution API",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSQLCommand(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.warehouseID, "warehouse-id", "", "ID of the SQL warehouse to use")
	cmd.Flags().StringVar(&opts.inlineSQL, "sql", "", "Inline SQL statement to execute")
	cmd.Flags().StringVar(&opts.file, "file", "", "Path to a file containing SQL statements")
	cmd.Flags().StringVar(&opts.rawFormat, "format", "", "Output format: table, json, or csv")
	cmd.Flags().DurationVar(&opts.waitTimeout, "wait-timeout", 10*time.Second, "Maximum time to wait synchronously before polling for results")
	cmd.Flags().StringVar(&opts.resultFile, "result-file", "", "Write query results to the specified file path")
	cmd.Flags().BoolVar(&opts.allowDestructiveFlag, "allow-destructive", false, "Allow statements that may have side effects")

	_ = cmd.MarkFlagRequired("warehouse-id")
	cmd.MarkFlagsMutuallyExclusive("sql", "file")
	cmd.MarkFlagsOneRequired("sql", "file")
	return cmd
}

func runSQLCommand(cmd *cobra.Command, opts *sqlOptions) error {
	script, err := loadSQLText(opts)
	if err != nil {
		return err
	}

	statements, err := sqlsafe.ParseStatements(script)
	if err != nil {
		return err
	}
	if len(statements) == 0 {
		return errors.New("no SQL statements to execute")
	}

	ctx := cmd.Context()
	allowDestructive, err := destructiveOverride.Resolve(ctx, cmdctx.ConfigUsed(ctx), cmd.Flags().Changed("allow-destructive"), opts.allowDestructiveFlag)
	if err != nil {
		return err
	}
	if !allowDestructive {
		classifier := sqlsafe.NewClassifier(sqlsafe.DefaultPolicy())
		if err := classifier.Check(statements); err != nil {
			return destructiveOverride.BlockedError(err)
		}
	}

	format, err := determineFormat(cmd, opts.rawFormat, opts.resultFile)
	if err != nil {
		return err
	}

	waitTimeout, err := normalizeWaitTimeout(opts.waitTimeout)
	if err != nil {
		return err
	}

	output, closeFn, err := selectOutput(cmd, format, opts.resultFile)
	if err != nil {
		return err
	}
	if closeFn != nil {
		defer closeFn()
	}

	client := cmdctx.WorkspaceClient(ctx)
	return executor.Run(executor.Options{
		Context:     ctx,
		Client:      client,
		Statements:  statements,
		WarehouseID: opts.warehouseID,
		WaitTimeout: waitTimeout,
		Format:      format,
		Output:      output,
		Stderr:      cmd.ErrOrStderr(),
		LogString:   cmdio.LogString,
	})
}

func loadSQLText(opts *sqlOptions) (string, error) {
	if opts.inlineSQL != "" {
		return opts.inlineSQL, nil
	}
	data, err := os.ReadFile(opts.file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func determineFormat(cmd *cobra.Command, raw, resultFile string) (sqlFormat, error) {
	if raw != "" {
		f := sqlFormat(strings.ToLower(raw))
		if !isValidFormat(f) {
			return "", fmt.Errorf("unknown format %q", raw)
		}
		return f, nil
	}
	if resultFile != "" {
		return formatJSON, nil
	}
	if cmdio.IsInteractive(cmd.Context()) {
		return formatTable, nil
	}
	return formatJSON, nil
}

func selectOutput(cmd *cobra.Command, format sqlFormat, resultFile string) (io.Writer, func(), error) {
	if resultFile == "" {
		return cmd.OutOrStdout(), nil, nil
	}
	if format == formatTable {
		return nil, nil, errors.New("table format cannot be written to --result-file; use --format json or csv")
	}
	file, err := createResultFile(resultFile)
	if err != nil {
		return nil, nil, err
	}
	return file, func() { _ = file.Close() }, nil
}

func isValidFormat(f sqlFormat) bool {
	switch f {
	case formatTable, formatJSON, formatCSV:
		return true
	default:
		return false
	}
}

func normalizeWaitTimeout(d time.Duration) (string, error) {
	if d < 0 {
		return "", errors.New("wait-timeout must be non-negative")
	}
	if d == 0 {
		return "0s", nil
	}
	if d%time.Second != 0 {
		return "", errors.New("wait-timeout must be specified in whole seconds")
	}
	seconds := int(d / time.Second)
	if seconds < 5 || seconds > 50 {
		return "", errors.New("wait-timeout must be between 5s and 50s, or 0s for async")
	}
	return fmt.Sprintf("%ds", seconds), nil
}

func createResultFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.Create(path)
}
