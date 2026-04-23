package validate_test

import (
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTags_AllTagsPresent(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
      tags:
        cost_center: "1234"
        data_owner: alpha
        classification: internal
  tag_validation_rules:
    r:
      securable_types: [catalog]
      required: [cost_center, data_owner, classification]
      allowed_values:
        classification: [public, internal, confidential]
`)
	diags := ucm.Apply(t.Context(), u, validate.ValidateTags())
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
}

func TestValidateTags_MissingRequired(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
      tags:
        cost_center: "1234"
  tag_validation_rules:
    r:
      securable_types: [catalog]
      required: [cost_center, data_owner, classification]
`)
	diags := ucm.Apply(t.Context(), u, validate.ValidateTags())
	require.Error(t, diags.Error())
	assert.Len(t, diags, 2) // data_owner + classification
	for _, d := range diags {
		assert.Equal(t, diag.Error, d.Severity)
	}
	assert.Contains(t, summaries(diags)[0], "classification")
	assert.Contains(t, summaries(diags)[1], "data_owner")
}

func TestValidateTags_DisallowedValue(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
      tags:
        cost_center: "1234"
        data_owner: alpha
        classification: bogus
  tag_validation_rules:
    r:
      securable_types: [catalog]
      required: [cost_center, data_owner, classification]
      allowed_values:
        classification: [public, internal, confidential]
`)
	diags := ucm.Apply(t.Context(), u, validate.ValidateTags())
	require.Error(t, diags.Error())
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "not in allowed values")
}

func TestValidateTags_SchemaSecurable(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  schemas:
    s1:
      name: s1
      catalog: c1
      # missing all required tags
  tag_validation_rules:
    r:
      securable_types: [schema]
      required: [data_owner]
`)
	diags := ucm.Apply(t.Context(), u, validate.ValidateTags())
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, `schema "s1"`)
}

// Catalogs of a type excluded from the rule must not be flagged.
func TestValidateTags_SecurableTypeFilter(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
      # no tags, but rule only applies to schemas
  tag_validation_rules:
    r:
      securable_types: [schema]
      required: [data_owner]
`)
	diags := ucm.Apply(t.Context(), u, validate.ValidateTags())
	assert.Empty(t, diags)
}

func TestValidateTags_NoRulesIsNoop(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
`)
	diags := ucm.Apply(t.Context(), u, validate.ValidateTags())
	assert.Empty(t, diags)
}

// Diagnostics must carry source locations pointing at the securable's tags
// map — this is what makes the errors jump-to-definition in editors.
func TestValidateTags_DiagnosticLocations(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
      tags:
        cost_center: "1234"
  tag_validation_rules:
    r:
      securable_types: [catalog]
      required: [data_owner]
`)
	diags := ucm.Apply(t.Context(), u, validate.ValidateTags())
	require.Len(t, diags, 1)
	require.NotEmpty(t, diags[0].Locations)
	assert.Equal(t, "/test/ucm.yml", diags[0].Locations[0].File)
}
