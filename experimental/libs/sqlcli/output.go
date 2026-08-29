// Package sqlcli holds patterns shared by experimental SQL-running commands
// (currently `experimental aitools tools query` and `experimental postgres
// query`). The package lives under experimental/libs/ rather than libs/ so
// the commands depending on it inherit experimental-stability guarantees:
// when both consumers graduate, this package can be promoted alongside
// (or its API stabilised first).
package sqlcli

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/env"
)

// EnvOutputFormat matches the env var name in cmd/root/io.go.
// Reading it lets pipelines set DATABRICKS_OUTPUT_FORMAT once for all
// commands.
const EnvOutputFormat = "DATABRICKS_OUTPUT_FORMAT"

// StaticTableThreshold is the row count above which interactive callers may
// hand off to libs/tableview's scrollable viewer. Smaller results stay in a
// static tabwriter table so they pipe to scripts unchanged.
const StaticTableThreshold = 30

// Format is the user-selectable output shape. Using a string typedef instead
// of an int enum keeps the help text and DATABRICKS_OUTPUT_FORMAT env var
// values self-describing.
type Format string

const (
	OutputText Format = "text"
	OutputJSON Format = "json"
	OutputCSV  Format = "csv"
)

// AllFormats is the canonical order shown in completions / help. Sharing
// the slice avoids drift between consumers when a new format is added.
var AllFormats = []Format{OutputText, OutputJSON, OutputCSV}

// ResolveFormat picks the effective output format. Precedence:
//
//  1. The local --output flag if it was explicitly set.
//  2. DATABRICKS_OUTPUT_FORMAT env var if set to a known value (invalid
//     values are silently ignored, matching cmd/root/io.go and aitools).
//  3. The flag default (whatever the caller passes as flagValue).
//
// Then the auto-selection rule applies: a *defaulted* text mode on a non-TTY
// stdout falls back to JSON, so scripts piping the output get machine-
// readable output by default. An *explicit* --output text (flag or env) is
// honoured even on a pipe; per AGENTS.md we don't silently override flags
// the user set.
//
// flagSet is true if the user explicitly passed --output on the CLI.
// stdoutTTY is true if stdout is a terminal.
func ResolveFormat(ctx context.Context, flagValue string, flagSet, stdoutTTY bool) (Format, error) {
	chosen := Format(strings.ToLower(flagValue))
	chosenExplicit := flagSet

	if !flagSet {
		if v, ok := env.Lookup(ctx, EnvOutputFormat); ok {
			candidate := Format(strings.ToLower(v))
			if IsKnown(candidate) {
				chosen = candidate
				chosenExplicit = true
			}
		}
	}

	if !IsKnown(chosen) {
		return "", fmt.Errorf("unsupported output format %q; expected one of: %s", flagValue, joinFormats(AllFormats))
	}

	if chosen == OutputText && !stdoutTTY && !chosenExplicit {
		return OutputJSON, nil
	}
	return chosen, nil
}

// IsKnown reports whether f is one of the formats in AllFormats.
func IsKnown(f Format) bool {
	return slices.Contains(AllFormats, f)
}

func joinFormats(formats []Format) string {
	parts := make([]string, len(formats))
	for i, f := range formats {
		parts[i] = string(f)
	}
	return strings.Join(parts, ", ")
}
