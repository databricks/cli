package dresources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFieldCopyValidate(t *testing.T) {
	copies := []interface{ Validate() error }{
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
	for _, c := range copies {
		require.NoError(t, c.Validate())
	}
}
