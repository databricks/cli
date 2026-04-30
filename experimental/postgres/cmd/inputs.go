package postgrescmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

// sqlFileExtension is the file suffix that triggers the .sql autodetect on a
// positional argument: if `databricks ... query foo.sql` exists on disk, we
// read it as a SQL file; otherwise it's treated as literal SQL.
const sqlFileExtension = ".sql"

// inputUnit is one SQL statement to execute, paired with metadata so the
// renderer can identify its origin in multi-input output shapes.
type inputUnit struct {
	// SQL is the trimmed statement text. Always non-empty by the time the
	// scanner has rejected multi-statement strings and empty inputs.
	SQL string
	// Source is a human-readable label for this input ("--file query.sql",
	// "stdin", or "argv[1]"). Used by the multi-input JSON renderer's "sql"
	// field hint and by the rich error formatter.
	Source string
}

// collectInputs assembles the ordered list of input units from positional
// arguments, --file flags, and stdin.
//
// Execution order is files-first then positionals (plan section "Statement
// execution"). Cobra/pflag does not preserve the user's interleaved CLI
// spelling: it collects all --file flags into one slice and all positionals
// into another, so we cannot honour `--file q1.sql "SELECT 1" --file q2.sql`
// as written. This is documented in --help.
//
// Stdin is read only when neither positional nor --file is provided.
func collectInputs(ctx context.Context, in io.Reader, args, files []string) ([]inputUnit, error) {
	var units []inputUnit

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read --file %q: %w", path, err)
		}
		sql := strings.TrimSpace(string(data))
		if sql == "" {
			return nil, fmt.Errorf("--file %q is empty", path)
		}
		units = append(units, inputUnit{SQL: sql, Source: "--file " + path})
	}

	for i, arg := range args {
		// .sql autodetect: if the positional ends in .sql AND the file
		// exists, read it as a SQL file. Other read errors (permission
		// denied) surface directly. If the file does not exist, fall through
		// and treat the positional as literal SQL — useful when the user
		// passes a string that happens to end with ".sql".
		if strings.HasSuffix(arg, sqlFileExtension) {
			data, err := os.ReadFile(arg)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("read positional %q: %w", arg, err)
			}
			if err == nil {
				sql := strings.TrimSpace(string(data))
				if sql == "" {
					return nil, fmt.Errorf("positional %q is empty", arg)
				}
				units = append(units, inputUnit{SQL: sql, Source: arg})
				continue
			}
		}
		sql := strings.TrimSpace(arg)
		if sql == "" {
			return nil, fmt.Errorf("argv[%d] is empty", i+1)
		}
		units = append(units, inputUnit{SQL: sql, Source: fmt.Sprintf("argv[%d]", i+1)})
	}

	if len(units) == 0 {
		// No positionals, no --file: read from stdin if it's not a prompt-
		// supporting TTY. The aitools query helper applies the same rule.
		_, isOsFile := in.(*os.File)
		if isOsFile && cmdio.IsPromptSupported(ctx) {
			return nil, errors.New("no SQL provided; pass a SQL string, use --file, or pipe via stdin")
		}
		data, err := io.ReadAll(in)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		sql := strings.TrimSpace(string(data))
		if sql == "" {
			return nil, errors.New("no SQL provided")
		}
		units = append(units, inputUnit{SQL: sql, Source: "stdin"})
	}

	return units, nil
}
