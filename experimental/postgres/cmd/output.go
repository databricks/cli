package postgrescmd

import (
	"context"
	"fmt"
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
// Then the auto-selection rule applies: text on a non-TTY stdout falls back
// to JSON. This matches the aitools query command and means scripts piping
// stdout get machine-readable output by default.
//
// flagSet is true if the user explicitly passed --output. stdoutTTY is true
// if stdout is a terminal.
func resolveOutputFormat(ctx context.Context, flagValue string, flagSet, stdoutTTY bool) (outputFormat, error) {
	chosen := outputFormat(strings.ToLower(flagValue))

	if !flagSet {
		if v, ok := env.Lookup(ctx, envOutputFormat); ok {
			candidate := outputFormat(strings.ToLower(v))
			if isKnownOutputFormat(candidate) {
				chosen = candidate
			}
		}
	}

	if !isKnownOutputFormat(chosen) {
		return "", fmt.Errorf("unsupported output format %q; expected one of: text, json, csv", flagValue)
	}

	if chosen == outputText && !stdoutTTY {
		return outputJSON, nil
	}
	return chosen, nil
}

func isKnownOutputFormat(f outputFormat) bool {
	switch f {
	case outputText, outputJSON, outputCSV:
		return true
	}
	return false
}
