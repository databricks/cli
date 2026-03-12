package root

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	unknownFlagPrefix          = "unknown flag: "
	unknownShorthandFlagPrefix = "unknown shorthand flag: "
	maxSuggestionDistance       = 2
)

// levenshteinDistance computes the edit distance between two strings.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Use a single row for the DP table.
	prev := make([]int, len(b)+1)
	for j := range len(b) + 1 {
		prev[j] = j
	}

	for i := range len(a) {
		curr := make([]int, len(b)+1)
		curr[0] = i + 1
		for j := range len(b) {
			cost := 1
			if a[i] == b[j] {
				cost = 0
			}
			curr[j+1] = min(
				curr[j]+1,     // insertion
				prev[j+1]+1,  // deletion
				prev[j]+cost, // substitution
			)
		}
		prev = curr
	}

	return prev[len(b)]
}

// suggestFlagFromError inspects the error message from Cobra for "unknown flag" patterns.
// If a close match is found among the command's flags, it returns an enhanced error
// with a "Did you mean" suggestion appended. Otherwise it returns the original error.
func suggestFlagFromError(cmd *cobra.Command, err error) error {
	msg := err.Error()

	if strings.HasPrefix(msg, unknownShorthandFlagPrefix) {
		return suggestShorthandFlag(cmd, err, msg)
	}

	if strings.HasPrefix(msg, unknownFlagPrefix) {
		return suggestLongFlag(cmd, err, msg)
	}

	return err
}

// suggestLongFlag suggests a matching long flag name for an "unknown flag: --xyz" error.
func suggestLongFlag(cmd *cobra.Command, original error, msg string) error {
	// Extract the flag name: "unknown flag: --flagname" -> "flagname"
	flagName := strings.TrimPrefix(msg, unknownFlagPrefix)
	flagName = strings.TrimLeft(flagName, "-")
	if flagName == "" {
		return original
	}

	best, bestDist := findClosestFlag(cmd, flagName)
	if best == "" || bestDist > maxSuggestionDistance {
		return original
	}

	return fmt.Errorf("%w\n\nDid you mean \"--%s\"?", original, best)
}

// suggestShorthandFlag suggests a matching shorthand for an
// "unknown shorthand flag: 'x' in -x" error.
func suggestShorthandFlag(cmd *cobra.Command, original error, msg string) error {
	// Extract the shorthand character: "unknown shorthand flag: 'x' in -x"
	rest := strings.TrimPrefix(msg, unknownShorthandFlagPrefix)
	if len(rest) < 3 || rest[0] != '\'' || rest[2] != '\'' {
		return original
	}
	ch := string(rest[1])

	best := findClosestShorthand(cmd, ch)
	if best == "" {
		return original
	}

	return fmt.Errorf("%w\n\nDid you mean \"-%s\"?", original, best)
}

// findClosestFlag returns the closest non-hidden, non-deprecated long flag name
// and its edit distance from the given misspelled name.
func findClosestFlag(cmd *cobra.Command, name string) (string, int) {
	best := ""
	bestDist := maxSuggestionDistance + 1

	seen := map[string]bool{}
	check := func(f *pflag.Flag) {
		if f.Hidden || f.Deprecated != "" {
			return
		}
		if seen[f.Name] {
			return
		}
		seen[f.Name] = true

		d := levenshteinDistance(name, f.Name)
		if d < bestDist {
			bestDist = d
			best = f.Name
		}
	}

	cmd.Flags().VisitAll(check)
	cmd.InheritedFlags().VisitAll(check)

	return best, bestDist
}

// findClosestShorthand returns the closest non-hidden, non-deprecated shorthand
// that differs by at most 1 edit from the given character.
func findClosestShorthand(cmd *cobra.Command, ch string) string {
	best := ""
	bestDist := maxSuggestionDistance + 1

	seen := map[string]bool{}
	check := func(f *pflag.Flag) {
		if f.Hidden || f.Deprecated != "" || f.Shorthand == "" {
			return
		}
		if seen[f.Shorthand] {
			return
		}
		seen[f.Shorthand] = true

		d := levenshteinDistance(ch, f.Shorthand)
		if d < bestDist {
			bestDist = d
			best = f.Shorthand
		}
	}

	cmd.Flags().VisitAll(check)
	cmd.InheritedFlags().VisitAll(check)

	return best
}
