package root

import (
	"errors"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseUnknownFlag triggers Cobra's flag parsing on args and returns the error.
// The command is set up with DisableFlagParsing=false (default) and a
// RunE that does nothing, so the only errors come from flag parsing.
func parseUnknownFlag(cmd *cobra.Command, args []string) error {
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
	cmd.SetArgs(args)
	return cmd.Execute()
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"kitten", "sitting", 3},
		{"output", "outpu", 1},   // deletion
		{"output", "ouptut", 2},  // transposition = 2 edits
		{"output", "outpux", 1},  // substitution
		{"output", "outputx", 1}, // insertion
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.a, tt.b), func(t *testing.T) {
			assert.Equal(t, tt.want, levenshteinDistance(tt.a, tt.b))
		})
	}
}

func TestSuggestFlagFromError_LongFlagCloseMatch(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("output", "", "output format")

	err := &pflag.NotExistError{}
	// Parse "--outpu" to get a real error
	parseErr := parseUnknownFlag(cmd, []string{"--outpu"})
	require.Error(t, parseErr)

	// Extract the pflag error from the cobra wrapping
	require.ErrorAs(t, parseErr, &err)

	got := suggestFlagFromError(cmd, parseErr)
	assert.Contains(t, got.Error(), `Did you mean "--output"?`)
	assert.Contains(t, got.Error(), "unknown flag: --outpu")
}

func TestSuggestFlagFromError_LongFlagNoMatch(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("output", "", "output format")

	parseErr := parseUnknownFlag(cmd, []string{"--zzzzzzz"})
	require.Error(t, parseErr)

	got := suggestFlagFromError(cmd, parseErr)
	assert.NotContains(t, got.Error(), "Did you mean")
}

func TestSuggestFlagFromError_ShorthandFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("output", "o", "", "output format")

	parseErr := parseUnknownFlag(cmd, []string{"-O"})
	require.Error(t, parseErr)

	got := suggestFlagFromError(cmd, parseErr)
	assert.Contains(t, got.Error(), `Did you mean "-o"?`)
}

func TestSuggestFlagFromError_HiddenFlagsExcluded(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("secret", "", "secret flag")
	require.NoError(t, cmd.Flags().MarkHidden("secret"))

	parseErr := parseUnknownFlag(cmd, []string{"--secre"})
	require.Error(t, parseErr)

	got := suggestFlagFromError(cmd, parseErr)
	assert.NotContains(t, got.Error(), "Did you mean")
}

func TestSuggestFlagFromError_DeprecatedFlagsExcluded(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("legacy", "", "old flag")
	require.NoError(t, cmd.Flags().MarkDeprecated("legacy", "use --new instead"))

	parseErr := parseUnknownFlag(cmd, []string{"--legac"})
	require.Error(t, parseErr)

	got := suggestFlagFromError(cmd, parseErr)
	assert.NotContains(t, got.Error(), "Did you mean")
}

func TestSuggestFlagFromError_InheritedFlags(t *testing.T) {
	parent := &cobra.Command{Use: "parent"}
	parent.PersistentFlags().String("profile", "", "auth profile")

	child := &cobra.Command{Use: "child"}
	child.RunE = func(cmd *cobra.Command, args []string) error { return nil }
	parent.AddCommand(child)

	parent.SetArgs([]string{"child", "--profil"})
	parseErr := parent.Execute()
	require.Error(t, parseErr)

	got := suggestFlagFromError(child, parseErr)
	assert.Contains(t, got.Error(), `Did you mean "--profile"?`)
}

func TestSuggestFlagFromError_NonFlagError(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("output", "", "output format")

	err := errors.New("some other error")
	got := suggestFlagFromError(cmd, err)
	assert.Equal(t, err.Error(), got.Error())
}

func TestSuggestFlagFromError_DeduplicatesLocalAndInherited(t *testing.T) {
	parent := &cobra.Command{Use: "parent"}
	parent.PersistentFlags().String("target", "", "deployment target")

	child := &cobra.Command{Use: "child"}
	child.Flags().String("target", "", "deployment target")
	child.RunE = func(cmd *cobra.Command, args []string) error { return nil }
	parent.AddCommand(child)

	parent.SetArgs([]string{"child", "--targe"})
	parseErr := parent.Execute()
	require.Error(t, parseErr)

	got := suggestFlagFromError(child, parseErr)
	assert.Contains(t, got.Error(), `Did you mean "--target"?`)
}

func TestSuggestFlagFromError_ShorthandUnrelatedNoSuggestion(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("output", "o", "", "output format")

	parseErr := parseUnknownFlag(cmd, []string{"-z"})
	require.Error(t, parseErr)

	got := suggestFlagFromError(cmd, parseErr)
	assert.NotContains(t, got.Error(), "Did you mean")
}

func TestSuggestFlagFromError_ShorthandDeprecatedStillSuggestsLongFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("output", "o", "", "output format")
	require.NoError(t, cmd.Flags().MarkShorthandDeprecated("output", "use --output instead"))

	parseErr := parseUnknownFlag(cmd, []string{"--outpu"})
	require.Error(t, parseErr)

	// The long flag should still be suggested even though the shorthand is deprecated.
	got := suggestFlagFromError(cmd, parseErr)
	assert.Contains(t, got.Error(), `Did you mean "--output"?`)
}

func TestSuggestFlagFromError_ShorthandDeprecatedExcludedFromShorthandSuggestions(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("output", "o", "", "output format")
	require.NoError(t, cmd.Flags().MarkShorthandDeprecated("output", "use --output instead"))

	parseErr := parseUnknownFlag(cmd, []string{"-O"})
	require.Error(t, parseErr)

	// The deprecated shorthand should NOT be suggested.
	got := suggestFlagFromError(cmd, parseErr)
	assert.NotContains(t, got.Error(), "Did you mean")
}

func TestSuggestFlagFromError_TieBreakingEquidistantFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	// "ab" and "ac" are both distance 1 from "aa"
	cmd.Flags().String("ab", "", "")
	cmd.Flags().String("ac", "", "")

	parseErr := parseUnknownFlag(cmd, []string{"--aa"})
	require.Error(t, parseErr)

	got := suggestFlagFromError(cmd, parseErr)
	// Both are equidistant; we accept whichever is returned (order depends on
	// flag iteration) but a suggestion must be present.
	assert.Contains(t, got.Error(), "Did you mean")
}

func TestSuggestFlagFromError_IntegrationThroughFlagErrorFunc(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("output", "", "output format")
	cmd.SetFlagErrorFunc(flagErrorFunc)
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
	cmd.SetArgs([]string{"--outpu"})

	err := cmd.Execute()
	require.Error(t, err)

	assert.Contains(t, err.Error(), `Did you mean "--output"?`)
	// flagErrorFunc also appends usage
	assert.Contains(t, err.Error(), "Usage:")
}
