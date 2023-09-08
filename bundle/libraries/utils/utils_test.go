package utils

import (
	"fmt"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/require"
)

var testCases map[string]bool = map[string]bool{
	"./some/local/path":          true,
	"/some/full/path":            true,
	"/Workspace/path/to/package": false,
	"/Users/path/to/package":     false,
	"file://path/to/package":     true,
	"C:\\path\\to\\package":      true,
	"dbfs://path/to/package":     false,
	"dbfs:/path/to/package":      false,
	"s3://path/to/package":       false,
	"abfss://path/to/package":    false,
}

func TestIsLocalLbrary(t *testing.T) {
	for p, result := range testCases {
		lib := compute.Library{
			Whl: p,
		}
		require.Equal(t, result, IsLocalLibrary(&lib), fmt.Sprintf("isLocalLibrary must return %t for path %s ", result, p))
	}
}
