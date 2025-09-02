package testdiff

import (
	"fmt"

	"github.com/databricks/cli/internal/testutil"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/stretchr/testify/assert"
)

func UnifiedDiff(filename1, filename2, s1, s2 string) string {
	edits := myers.ComputeEdits(span.URIFromPath(filename1), s1, s2)
	return fmt.Sprint(gotextdiff.ToUnified(filename1, filename2, s1, edits))
}

func AssertEqualTexts(t testutil.TestingT, filename1, filename2, expected, out string) bool {
	t.Helper()
	if len(out) < 1000 && len(expected) < 1000 {
		// This shows full strings + diff which could be useful when debugging newlines
		return assert.Equal(t, expected, out, "%s vs %s", filename1, filename2)
	} else {
		// only show diff for large texts
		diff := UnifiedDiff(filename1, filename2, expected, out)
		if diff != "" {
			t.Error("Diff:\n" + diff)
			return false
		}
	}
	return true
}
