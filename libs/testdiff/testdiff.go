package testdiff

import (
	"fmt"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/stretchr/testify/assert"
	"github.com/wI2L/jsondiff"
)

func UnifiedDiff(filename1, filename2, s1, s2 string) string {
	edits := myers.ComputeEdits(span.URIFromPath(filename1), s1, s2)
	return fmt.Sprint(gotextdiff.ToUnified(filename1, filename2, s1, edits))
}

func AssertEqualTexts(t testutil.TestingT, filename1, filename2, expected, out string) {
	if len(out) < 1000 && len(expected) < 1000 {
		// This shows full strings + diff which could be useful when debugging newlines
		assert.Equal(t, expected, out)
	} else {
		// only show diff for large texts
		diff := UnifiedDiff(filename1, filename2, expected, out)
		t.Errorf("Diff:\n" + diff)
	}
}

func AssertEqualJSONs(t *testing.T, expectedName, outName, expected, out string, ignorePaths []string) {
	patch, err := jsondiff.CompareJSON([]byte(expected), []byte(out))
	if err != nil {
		t.Logf("CompareJSON error for %s vs %s: %s (fallback to textual comparison)", outName, expectedName, err)
		AssertEqualTexts(t, expectedName, outName, expected, out)
	} else {
		diff := UnifiedDiff(expectedName, outName, expected, out)
		t.Logf("Diff:\n%s", diff)
		ignoredDiffs := []string{}
		erroredDiffs := []string{}
		for _, op := range patch {
			if matchesPrefixes(ignorePaths, op.Path) {
				ignoredDiffs = append(ignoredDiffs, fmt.Sprintf("%7s %s %v", op.Type, op.Path, op.Value))
			} else {
				erroredDiffs = append(erroredDiffs, fmt.Sprintf("%7s %s %v", op.Type, op.Path, op.Value))
			}
		}
		if len(ignoredDiffs) > 0 {
			t.Logf("Ignored differences between %s and %s:\n ==> %s", expectedName, outName, strings.Join(ignoredDiffs, "\n ==> "))
		}
		if len(erroredDiffs) > 0 {
			t.Errorf("Unexpected differences between %s and %s:\n ==> %s", expectedName, outName, strings.Join(erroredDiffs, "\n ==> "))
		}
	}
}

func matchesPrefixes(prefixes []string, path string) bool {
	for _, p := range prefixes {
		if p == path {
			return true
		}
		if strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}
