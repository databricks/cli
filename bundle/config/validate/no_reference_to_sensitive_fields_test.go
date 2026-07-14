package validate

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sensitiveTestResource is a minimal struct with one sensitive field, used to
// exercise the validator without a real resource that carries `bundle:"sensitive"`.
type sensitiveTestResource struct {
	Name  string `json:"name"`
	Token string `json:"token" bundle:"sensitive"`
}

// withSensitiveResourceType registers sensitiveTestResource under the key
// "secrets" in config.ResourcesTypes for the duration of the test.
func withSensitiveResourceType(t *testing.T) {
	t.Helper()
	orig := config.ResourcesTypes
	patched := make(map[string]reflect.Type, len(orig)+1)
	for k, v := range orig {
		patched[k] = v
	}
	patched["secrets"] = reflect.TypeFor[sensitiveTestResource]()
	config.ResourcesTypes = patched
	t.Cleanup(func() { config.ResourcesTypes = orig })
}

func makeJobsBundle(t *testing.T) *bundle.Bundle {
	t.Helper()
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"src": {JobSettings: jobs.JobSettings{Name: "source"}},
					"dst": {JobSettings: jobs.JobSettings{Name: "placeholder"}},
				},
			},
		},
	}
}

func TestNoReferenceToSensitiveFields_NoSensitiveType(t *testing.T) {
	// jobs has no sensitive fields; a cross-resource reference is allowed.
	b := makeJobsBundle(t)
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("${resources.jobs.src.name}"))
	})
	diags := NoReferenceToSensitiveFields().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestNoReferenceToSensitiveFields_NonResourceReference(t *testing.T) {
	b := makeJobsBundle(t)
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("${bundle.name}"))
	})
	diags := NoReferenceToSensitiveFields().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestNoReferenceToSensitiveFields_ShortPath(t *testing.T) {
	// Reference stops before the field component — must not panic or error.
	b := makeJobsBundle(t)
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("${resources.jobs.src}"))
	})
	diags := NoReferenceToSensitiveFields().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestNoReferenceToSensitiveFields_NonSensitiveFieldOnSensitiveType(t *testing.T) {
	withSensitiveResourceType(t)
	b := makeJobsBundle(t)
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("${resources.secrets.my_secret.name}"))
	})
	diags := NoReferenceToSensitiveFields().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestNoReferenceToSensitiveFields_SensitiveFieldReference(t *testing.T) {
	withSensitiveResourceType(t)
	b := makeJobsBundle(t)
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("${resources.secrets.my_secret.token}"))
	})
	diags := NoReferenceToSensitiveFields().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "resources.secrets.my_secret.token")
	assert.Contains(t, diags[0].Summary, "sensitive field")
}

func TestNoReferenceToSensitiveFields_SensitiveFieldInJobName(t *testing.T) {
	// Sensitive reference inside a non-sensitive field of another resource is also caught.
	withSensitiveResourceType(t)
	b := makeJobsBundle(t)
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("prefix-${resources.secrets.my_secret.token}-suffix"))
	})
	diags := NoReferenceToSensitiveFields().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "resources.secrets.my_secret.token")
}
