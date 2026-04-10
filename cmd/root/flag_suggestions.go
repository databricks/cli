package root

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const maxSuggestionDistance = 2

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
				curr[j]+1,    // insertion
				prev[j+1]+1,  // deletion
				prev[j]+cost, // substitution
			)
		}
		prev = curr
	}

	return prev[len(b)]
}

// suggestFlagFromError inspects the error from Cobra for unknown-flag errors.
// If a close match is found among the command's flags, it returns an enhanced error
// with a "Did you mean" suggestion appended. Otherwise it returns the original error.
func suggestFlagFromError(cmd *cobra.Command, err error) error {
	var notExist *pflag.NotExistError
	if !errors.As(err, &notExist) {
		return err
	}

	flagName := notExist.GetSpecifiedName()
	isShorthand := notExist.GetSpecifiedShortnames() != ""

	if isShorthand {
		return suggestShorthandFlag(cmd, err, flagName)
	}

	return suggestLongFlag(cmd, err, flagName)
}

// suggestLongFlag suggests a matching long flag name for an unknown long flag error.
func suggestLongFlag(cmd *cobra.Command, original error, flagName string) error {
	if flagName == "" {
		return original
	}

	best, bestDist := findClosestFlag(cmd, flagName)
	if best == "" || bestDist > maxSuggestionDistance {
		return original
	}

	return fmt.Errorf("%w\n\nDid you mean \"--%s\"?", original, best)
}

// suggestShorthandFlag suggests a matching shorthand for an unknown shorthand flag error.
func suggestShorthandFlag(cmd *cobra.Command, original error, flagName string) error {
	if flagName == "" {
		return original
	}
	ch := string(flagName[0])

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

// findClosestShorthand returns a case-insensitive exact match for the given
// shorthand character. Levenshtein is not useful for single characters because
// any two distinct characters always have distance 1.
func findClosestShorthand(cmd *cobra.Command, ch string) string {
	best := ""
	seen := map[string]bool{}
	check := func(f *pflag.Flag) {
		if f.Hidden || f.Deprecated != "" || f.ShorthandDeprecated != "" || f.Shorthand == "" {
			return
		}
		if seen[f.Shorthand] {
			return
		}
		seen[f.Shorthand] = true
		if strings.EqualFold(ch, f.Shorthand) {
			best = f.Shorthand
		}
	}
	cmd.Flags().VisitAll(check)
	cmd.InheritedFlags().VisitAll(check)
	return best
}
