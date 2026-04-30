package postgrescmd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/env"
)

// outputFormat is the user-selectable output shape. Using a string typedef
// instead of an int enum keeps the help text and DATABRICKS_OUTPUT_FORMAT env
// var values self-describing.
type outputFormat string

const (
	outputText outputFormat = "text"
	outputJSON outputFormat = "json"
	outputCSV  outputFormat = "csv"

	// envOutputFormat matches the env var name in cmd/root/io.go. Reading it
	// here lets pipelines set DATABRICKS_OUTPUT_FORMAT once for all
	// commands. See aitools query for a parallel pattern.
	envOutputFormat = "DATABRICKS_OUTPUT_FORMAT"
)

// allOutputFormats is the canonical order shown in completions / help.
var allOutputFormats = []outputFormat{outputText, outputJSON, outputCSV}

// resolveOutputFormat picks the effective output format. Precedence:
//
//  1. The local --output flag if it was explicitly set.
//  2. DATABRICKS_OUTPUT_FORMAT env var if set to a known value (invalid
//     values are silently ignored, matching cmd/root/io.go and aitools).
//  3. The flag default ("text").
//
// Then the auto-selection rule applies: a *defaulted* text mode on a non-TTY
// stdout falls back to JSON, so scripts piping the output get machine-
// readable output by default. An *explicit* --output text is honoured even
// on a pipe; per CLAUDE.md we don't silently override flags the user set.
//
// flagSet is true if the user explicitly passed --output. stdoutTTY is true
// if stdout is a terminal.
func resolveOutputFormat(ctx context.Context, flagValue string, flagSet, stdoutTTY bool) (outputFormat, error) {
	chosen := outputFormat(strings.ToLower(flagValue))
	chosenExplicit := flagSet

	if !flagSet {
		if v, ok := env.Lookup(ctx, envOutputFormat); ok {
			candidate := outputFormat(strings.ToLower(v))
			if isKnownOutputFormat(candidate) {
				chosen = candidate
				chosenExplicit = true
			}
		}
	}

	if !isKnownOutputFormat(chosen) {
		return "", fmt.Errorf("unsupported output format %q; expected one of: %s", flagValue, joinOutputFormats(allOutputFormats))
	}

	if chosen == outputText && !stdoutTTY && !chosenExplicit {
		return outputJSON, nil
	}
	return chosen, nil
}

func joinOutputFormats(formats []outputFormat) string {
	parts := make([]string, len(formats))
	for i, f := range formats {
		parts[i] = string(f)
	}
	return strings.Join(parts, ", ")
}

func isKnownOutputFormat(f outputFormat) bool {
	return slices.Contains(allOutputFormats, f)
}
