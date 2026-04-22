package dresources

import (
	"reflect"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// TestPipelineRemote verifies that all fields from pipelines.GetPipelineResponse
// (except Spec and internal fields) are present in PipelineRemote.
func TestPipelineRemote(t *testing.T) {
	assertFieldsCovered(t, reflect.TypeFor[pipelines.GetPipelineResponse](), reflect.TypeFor[PipelineRemote](), map[string]bool{
		"Spec":            true, // Embedded as pipelines.CreatePipeline (via makePipelineRemote)
		"ForceSendFields": true, // Internal marshaling field
		"Name":            true, // Available through embedded CreatePipeline
		"RunAs":           true, // Available through embedded CreatePipeline
	})
}
