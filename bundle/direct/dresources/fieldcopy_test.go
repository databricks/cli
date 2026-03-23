package dresources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFieldCopyValidate(t *testing.T) {
	copies := []interface{ Validate() error }{
		&clusterRemapCopy,
		&clusterCreateCopy,
		&clusterEditCopy,
	}
	for _, c := range copies {
		require.NoError(t, c.Validate())
	}
}
