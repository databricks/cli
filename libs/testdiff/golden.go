package testdiff

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

var OverwriteMode = false

func init() {
	flag.BoolVar(&OverwriteMode, "update", false, "Overwrite golden files")
}

func ReadFile(t testutil.TestingT, ctx context.Context, filename string) string {
	t.Helper()
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return ""
	}
	assert.NoError(t, err, "Failed to read %s", filename)
	// On CI, on Windows \n in the file somehow end up as \r\n
	return NormalizeNewlines(string(data))
}

func WriteFile(t testutil.TestingT, filename, data string) {
	t.Helper()
	t.Logf("Overwriting %s", filename)
	err := os.WriteFile(filename, []byte(data), 0o644)
	assert.NoError(t, err, "Failed to write %s", filename)
}

func AssertOutput(t testutil.TestingT, ctx context.Context, out, outTitle, expectedPath string) {
	t.Helper()
	expected := ReadFile(t, ctx, expectedPath)

	out = ReplaceOutput(t, ctx, out)

	if out != expected {
		AssertEqualTexts(t, expectedPath, outTitle, expected, out)

		if OverwriteMode {
			WriteFile(t, expectedPath, out)
		}
	}
}

func ReplaceOutput(t testutil.TestingT, ctx context.Context, out string) string {
	t.Helper()
	out = NormalizeNewlines(out)
	replacements := GetReplacementsMap(ctx)
	if replacements == nil {
		t.Fatal("WithReplacementsMap was not called")
	}
	return replacements.Replace(out)
}

func NormalizeNewlines(input string) string {
	output := strings.ReplaceAll(input, "\r\n", "\n")
	return strings.ReplaceAll(output, "\r", "\n")
}
