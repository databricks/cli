package dresources

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testdiff"
)

func TestFieldCopyReport(t *testing.T) {
	ctx := context.Background()
	ctx, _ = testdiff.WithReplacementsMap(ctx)

	copies := []interface{ Report() string }{
		// cluster
		&clusterRemapCopy,
		&clusterCreateCopy,
		&clusterEditCopy,
		// job
		&jobCreateCopy,
		// pipeline
		&pipelineSpecCopy,
		&pipelineRemoteCopy,
		&pipelineEditCopy,
		// model serving endpoint
		&autoCaptureConfigCopy,
		&servedEntityCopy,
		&servingRemapCopy,
	}

	var buf strings.Builder
	for _, c := range copies {
		buf.WriteString(c.Report())
	}

	goldenPath := filepath.Join("testdata", "out.fieldcopy.txt")
	testdiff.AssertOutput(t, ctx, buf.String(), "fieldcopy report", goldenPath)
}
