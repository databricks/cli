package root

import (
	"errors"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

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

	err := errors.New("unknown flag: --outpu")
	got := suggestFlagFromError(cmd, err)
	assert.Contains(t, got.Error(), `Did you mean "--output"?`)
	assert.Contains(t, got.Error(), "unknown flag: --outpu")
}

func TestSuggestFlagFromError_LongFlagNoMatch(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("output", "", "output format")

	err := errors.New("unknown flag: --zzzzzzz")
	got := suggestFlagFromError(cmd, err)
	assert.Equal(t, err.Error(), got.Error())
}

func TestSuggestFlagFromError_ShorthandFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("output", "o", "", "output format")

	err := errors.New("unknown shorthand flag: 'O' in -O")
	got := suggestFlagFromError(cmd, err)
	assert.Contains(t, got.Error(), `Did you mean "-o"?`)
}

func TestSuggestFlagFromError_HiddenFlagsExcluded(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("secret", "", "secret flag")
	_ = cmd.Flags().MarkHidden("secret")

	err := errors.New("unknown flag: --secre")
	got := suggestFlagFromError(cmd, err)
	assert.NotContains(t, got.Error(), "Did you mean")
}

func TestSuggestFlagFromError_DeprecatedFlagsExcluded(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("legacy", "", "old flag")
	_ = cmd.Flags().MarkDeprecated("legacy", "use --new instead")

	err := errors.New("unknown flag: --legac")
	got := suggestFlagFromError(cmd, err)
	assert.NotContains(t, got.Error(), "Did you mean")
}

func TestSuggestFlagFromError_InheritedFlags(t *testing.T) {
	parent := &cobra.Command{Use: "parent"}
	parent.PersistentFlags().String("profile", "", "auth profile")

	child := &cobra.Command{Use: "child"}
	parent.AddCommand(child)

	err := errors.New("unknown flag: --profil")
	got := suggestFlagFromError(child, err)
	assert.Contains(t, got.Error(), `Did you mean "--profile"?`)
}

func TestSuggestFlagFromError_NonFlagError(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("output", "", "output format")

	err := errors.New("flag needs an argument: --output")
	got := suggestFlagFromError(cmd, err)
	assert.Equal(t, err.Error(), got.Error())
}

func TestSuggestFlagFromError_CobraErrorFormats(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		flags    map[string]string
		contains string
	}{
		{
			name:     "long flag with double dash",
			errMsg:   "unknown flag: --outpu",
			flags:    map[string]string{"output": ""},
			contains: `"--output"`,
		},
		{
			name:     "shorthand with no matching flags",
			errMsg:   "unknown shorthand flag: 'x' in -x",
			flags:    map[string]string{},
			contains: "unknown shorthand flag: 'x' in -x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			for name, usage := range tt.flags {
				cmd.Flags().String(name, "", usage)
			}
			err := errors.New(tt.errMsg)
			got := suggestFlagFromError(cmd, err)
			assert.Contains(t, got.Error(), tt.contains)
		})
	}
}

func TestSuggestFlagFromError_DeduplicatesLocalAndInherited(t *testing.T) {
	parent := &cobra.Command{Use: "parent"}
	parent.PersistentFlags().String("target", "", "deployment target")

	child := &cobra.Command{Use: "child"}
	child.Flags().String("target", "", "deployment target")
	parent.AddCommand(child)

	err := errors.New("unknown flag: --targe")
	got := suggestFlagFromError(child, err)

	// Should suggest once, not panic or produce duplicate suggestions.
	assert.Contains(t, got.Error(), `Did you mean "--target"?`)
}

func TestSuggestFlagFromError_EmptyFlagName(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("output", "", "output format")
	err := errors.New("unknown flag: --")
	got := suggestFlagFromError(cmd, err)
	assert.Equal(t, err.Error(), got.Error())
}

func TestSuggestFlagFromError_ShorthandUnrelatedNoSuggestion(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("output", "o", "", "output format")

	err := errors.New("unknown shorthand flag: 'z' in -z")
	got := suggestFlagFromError(cmd, err)
	assert.NotContains(t, got.Error(), "Did you mean")
	assert.Equal(t, err.Error(), got.Error())
}
