package dresources

import (
	"context"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testdiff"
)

func TestFieldCopyReport(t *testing.T) {
	ctx := context.Background()
	ctx, _ = testdiff.WithReplacementsMap(ctx)

	var buf strings.Builder
	for _, c := range allFieldCopies {
		buf.WriteString(c.Report())
	}

	testdiff.AssertOutput(t, ctx, buf.String(), "fieldcopy report", "out.fieldcopy.txt")
}
