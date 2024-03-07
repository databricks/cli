package mutator

import (
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

// Every new resource type added to DABs should either be classified in the allow
// list or the deny list for run_as other support.
func TestAllResourceTypesAreClassifiedForRunAs(t *testing.T) {
	// Compute supported resource types based on the `Resources{}` struct.
	r := config.Resources{}
	rv, err := convert.FromTyped(r, dyn.NilValue)
	require.NoError(t, err)
	normalized, _ := convert.Normalize(r, rv, convert.IncludeMissingFields)
	resourceTypes := maps.Keys(normalized.MustMap())
	slices.Sort(resourceTypes)

	// Assert that all resource types are classified in either the allow list or the deny list.
	for _, resourceType := range resourceTypes {
		if slices.Contains(allowListForRunAsOther, resourceType) && !slices.Contains(denyListForRunAsOther, resourceType) {
			continue
		}
		if !slices.Contains(allowListForRunAsOther, resourceType) && slices.Contains(denyListForRunAsOther, resourceType) {
			continue
		}
		if slices.Contains(allowListForRunAsOther, resourceType) && slices.Contains(denyListForRunAsOther, resourceType) {
			t.Errorf("Resource type %s is classified in both allow list and deny list for run_as other support", resourceType)
		}
		if !slices.Contains(allowListForRunAsOther, resourceType) && !slices.Contains(denyListForRunAsOther, resourceType) {
			t.Errorf("Resource type %s is not classified in either allow list or deny list for run_as other support", resourceType)
		}
	}

	// Assert the total list of resource supported, as a sanity check that using
	// the dyn library gives us the correct list of all resources supported. Please
	// also update this check when adding a new resource
	assert.Equal(t, []string{"experiments", "jobs", "model_serving_endpoints", "models", "pipelines", "registered_models"}, resourceTypes)
}
