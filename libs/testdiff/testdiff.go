package testdiff

import (
	"fmt"
	"strings"

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

func AssertEqualJQ(t testutil.TestingT, expectedName, outName, expected, out string, ignorePaths []string) {
	t.Helper()
	patch, err := jsondiff.CompareJSON([]byte(expected), []byte(out))
	if err != nil {
		t.Logf("CompareJSON error for %s vs %s: %s (fallback to textual comparison)", outName, expectedName, err)
		AssertEqualTexts(t, expectedName, outName, expected, out)
	} else {
		diff := UnifiedDiff(expectedName, outName, expected, out)
		t.Logf("Diff:\n%s", diff)
		allowedDiffs := []string{}
		erroredDiffs := []string{}
		for _, op := range patch {
			if allowDifference(ignorePaths, op) {
				allowedDiffs = append(allowedDiffs, fmt.Sprintf("%7s %s %v old=%v", op.Type, op.Path, op.Value, op.OldValue))
			} else {
				erroredDiffs = append(erroredDiffs, fmt.Sprintf("%7s %s %v old=%v", op.Type, op.Path, op.Value, op.OldValue))
			}
		}
		if len(allowedDiffs) > 0 {
			t.Logf("Allowed differences between %s and %s:\n ==> %s", expectedName, outName, strings.Join(allowedDiffs, "\n ==> "))
		}
		if len(erroredDiffs) > 0 {
			t.Errorf("Unexpected differences between %s and %s:\n ==> %s", expectedName, outName, strings.Join(erroredDiffs, "\n ==> "))
		}
	}
}

func allowDifference(ignorePaths []string, op jsondiff.Operation) bool {
	if matchesPrefixes(ignorePaths, op.Path) {
		return true
	}
	if op.Type == "replace" && almostSameStrings(op.OldValue, op.Value) {
		return true
	}
	return false
}

// compare strings and ignore forward vs backward slashes
func almostSameStrings(v1, v2 any) bool {
	s1, ok := v1.(string)
	if !ok {
		return false
	}
	s2, ok := v2.(string)
	if !ok {
		return false
	}
	return strings.ReplaceAll(s1, "\\", "/") == strings.ReplaceAll(s2, "\\", "/")
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
