package testutil

import (
	"fmt"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

func Diff(filename1, filename2, s1, s2 string) string {
	edits := myers.ComputeEdits(span.URIFromPath(filename1), s1, s2)
	return fmt.Sprint(gotextdiff.ToUnified(filename1, filename2, s1, edits))
}
