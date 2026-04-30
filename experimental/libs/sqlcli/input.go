package sqlcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

// SQLFileExtension is the file suffix that triggers the .sql autodetect on a
// positional argument: if `databricks ... query foo.sql` exists on disk, the
// argument is read as a SQL file; otherwise it's treated as literal SQL.
const SQLFileExtension = ".sql"

// Input is one SQL statement to execute, paired with a label identifying its
// origin so multi-input renderers and error messages can refer back to "which
// of the N inputs failed".
type Input struct {
	// SQL is the cleaned statement text. Always non-empty (Collect rejects
	// inputs that clean to empty).
	SQL string
	// Source is a human-readable label: "--file PATH", "argv[N]", or "stdin".
	Source string
}

// CollectOptions controls per-command behavior. The zero value is fine for
// commands that just want plain trimmed input.
type CollectOptions struct {
	// Cleaner is applied to each raw SQL after read (and before the empty
	// check). The default is strings.TrimSpace; aitools passes a richer
	// cleaner that strips SQL comments and surrounding quotes. Postgres
	// passes the default because its multi-statement scanner needs comments
	// preserved.
	Cleaner func(string) string
}

// Collect assembles the ordered list of inputs from --file paths, positional
// arguments, and stdin.
//
// Order is files-first, then positionals. Cobra/pflag does not preserve the
// user's interleaved CLI spelling: it collects all --file flags into one
// slice and all positionals into another, so callers cannot honour
// `--file q1.sql "SELECT 1" --file q2.sql` as written.
//
// Stdin is read only when neither --file nor positional input was provided,
// and only when stdin is not a prompt-capable TTY (otherwise we'd block
// waiting for input the user did not realise they had to type).
//
// Errors when:
//   - A --file path can't be read or cleans to empty.
//   - A positional that looks like a .sql file but read fails with a non-
//     "does not exist" error (e.g. permission denied).
//   - A positional cleans to empty.
//   - Stdin is the only source and it's empty / blocked on a TTY.
func Collect(ctx context.Context, in io.Reader, args, files []string, opts CollectOptions) ([]Input, error) {
	cleaner := opts.Cleaner
	if cleaner == nil {
		cleaner = strings.TrimSpace
	}

	var inputs []Input

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read --file %q: %w", path, err)
		}
		sql := cleaner(string(data))
		if sql == "" {
			return nil, fmt.Errorf("--file %q is empty", path)
		}
		inputs = append(inputs, Input{SQL: sql, Source: "--file " + path})
	}

	for i, arg := range args {
		// .sql autodetect: if the positional ends in .sql AND the file
		// exists, read it as a SQL file. Other read errors (permission
		// denied) surface directly. If the file does not exist, fall
		// through and treat the positional as literal SQL — useful when
		// the user passes a string that happens to end with ".sql".
		if strings.HasSuffix(arg, SQLFileExtension) {
			data, err := os.ReadFile(arg)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("read positional %q: %w", arg, err)
			}
			if err == nil {
				sql := cleaner(string(data))
				if sql == "" {
					return nil, fmt.Errorf("positional %q is empty", arg)
				}
				inputs = append(inputs, Input{SQL: sql, Source: arg})
				continue
			}
		}
		sql := cleaner(arg)
		if sql == "" {
			return nil, fmt.Errorf("argv[%d] is empty", i+1)
		}
		inputs = append(inputs, Input{SQL: sql, Source: fmt.Sprintf("argv[%d]", i+1)})
	}

	if len(inputs) == 0 {
		_, isOsFile := in.(*os.File)
		if isOsFile && cmdio.IsPromptSupported(ctx) {
			return nil, errors.New("no SQL provided; pass a SQL string, use --file, or pipe via stdin")
		}
		data, err := io.ReadAll(in)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		sql := cleaner(string(data))
		if sql == "" {
			return nil, errors.New("no SQL provided")
		}
		inputs = append(inputs, Input{SQL: sql, Source: "stdin"})
	}

	return inputs, nil
}
