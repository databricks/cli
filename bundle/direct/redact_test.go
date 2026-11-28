package direct

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secretLiteral = "dummy-credential"

func secretAdapter(t *testing.T) *dresources.Adapter {
	t.Helper()
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)
	adapter, ok := adapters["secrets"]
	require.True(t, ok)
	return adapter
}

// syncedRoot gives a Go-literal config the dynamic value that a loaded bundle has, which GetResourceConfig reads.
func syncedRoot(t *testing.T, root *config.Root) *config.Root {
	t.Helper()
	require.NoError(t, root.Mutate(func(v dyn.Value) (dyn.Value, error) { return v, nil }))
	return root
}

// TestRedactPlanEntryRedactsNewStateAndChanges checks that the literal is absent from the new_state
// and changes of a plan entry, and that the non-sensitive fields are kept.
func TestRedactPlanEntryRedactsNewStateAndChanges(t *testing.T) {
	adapter := secretAdapter(t)

	newState := &structvar.StructVarJSON{}
	var err error
	newState.Value, err = json.Marshal(map[string]any{
		"name":         "s",
		"catalog_name": "main",
		"schema_name":  "default",
		"value":        secretLiteral,
	})
	require.NoError(t, err)

	entry := &deployplan.PlanEntry{
		Action:   deployplan.Update,
		NewState: newState,
		Changes: deployplan.Changes{
			"value": {Action: deployplan.Update, Old: "", New: secretLiteral, Remote: secretLiteral},
			"name":  {Action: deployplan.Skip, Old: "s", New: "s", Remote: "s"},
		},
	}

	require.NoError(t, redactPlanEntry(adapter, entry))

	out, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.NotContains(t, string(out), secretLiteral)

	var redacted struct {
		NewState struct {
			Value map[string]any `json:"value"`
		} `json:"new_state"`
	}
	require.NoError(t, json.Unmarshal(out, &redacted))
	assert.Equal(t, "[redacted]", redacted.NewState.Value["value"])
	assert.Equal(t, "main", redacted.NewState.Value["catalog_name"])
	assert.Equal(t, "[redacted]", entry.Changes["value"].New)
	assert.Equal(t, "s", entry.Changes["name"].New)
}

func secretRoot(t *testing.T, value string, variables map[string]*variable.Variable) *config.Root {
	t.Helper()
	root := &config.Root{
		Variables: variables,
		Resources: config.Resources{
			Secrets: map[string]*resources.Secret{
				"s": {Secret: catalog.Secret{Name: "s", Value: value}},
			},
		},
	}
	return syncedRoot(t, root)
}

const (
	msgValueMissing = "resource resources.secrets.s: the config has no value for sensitive field value; set the variable it uses with --var or BUNDLE_VAR_<name>"
	msgNotInConfig  = "resource resources.secrets.gone is not in the config; run 'bundle plan' again"
)

// TestRestoreSensitiveFields checks the value a saved plan's sensitive field is restored to: what a
// plain deploy sends from the same config. A reference is kept for apply to resolve, and an escaped
// reference is sent as the literal it escapes.
func TestRestoreSensitiveFields(t *testing.T) {
	adapter := secretAdapter(t)

	tests := []struct {
		name        string
		resourceKey string
		planned     string
		configValue string
		wantValue   string
		wantRefs    map[string]string
		wantErr     string
	}{
		{
			name:        "redaction marker is replaced by the config value",
			resourceKey: "resources.secrets.s",
			planned:     sensitiveRedactedValue,
			configValue: secretLiteral,
			wantValue:   secretLiteral,
		},
		{
			name:        "plaintext from a plan saved by an older CLI is replaced by the config value",
			resourceKey: "resources.secrets.s",
			planned:     "stale-plaintext",
			configValue: secretLiteral,
			wantValue:   secretLiteral,
		},
		{
			name:        "zero in the plan but set in the config is restored",
			resourceKey: "resources.secrets.s",
			planned:     "",
			configValue: secretLiteral,
			wantValue:   secretLiteral,
		},
		{
			name:        "zero in both is skipped",
			resourceKey: "resources.secrets.s",
			planned:     "",
			configValue: "",
			wantValue:   "",
		},
		{
			name:        "non-zero in the plan with an empty config value is an error",
			resourceKey: "resources.secrets.s",
			planned:     sensitiveRedactedValue,
			configValue: "",
			wantErr:     msgValueMissing,
		},
		{
			name:        "non-zero in the plan with the resource missing from config is an error",
			resourceKey: "resources.secrets.gone",
			planned:     sensitiveRedactedValue,
			configValue: secretLiteral,
			wantErr:     msgNotInConfig,
		},
		{
			name:        "a reference in the config is kept for apply to resolve",
			resourceKey: "resources.secrets.s",
			planned:     sensitiveRedactedValue,
			configValue: "${resources.catalogs.c.name}",
			wantValue:   "${resources.catalogs.c.name}",
			wantRefs:    map[string]string{"value": "${resources.catalogs.c.name}"},
		},
		{
			name:        "an escaped reference is sent as the literal it escapes",
			resourceKey: "resources.secrets.s",
			planned:     sensitiveRedactedValue,
			configValue: "pre-$${x}",
			wantValue:   "pre-${x}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := secretRoot(t, tc.configValue, nil)
			planned := &structvar.StructVar{Value: &catalog.Secret{Name: "s", Value: tc.planned}}
			err := restoreSensitiveFields(adapter, tc.resourceKey, planned, root)
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			secret := planned.Value.(*catalog.Secret)
			assert.Equal(t, tc.wantValue, secret.Value)
			assert.Empty(t, secret.EffectiveValue)
			assert.Equal(t, tc.wantRefs, planned.Refs)
		})
	}
}

func TestCheckSensitivePlanValues(t *testing.T) {
	sensitive := true
	variables := map[string]*variable.Variable{
		"api_credential": {Sensitive: &sensitive},
		"target_schema":  {},
	}
	secretPlan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.secrets.s": {NewState: &structvar.StructVarJSON{Value: []byte(`{"name":"s","value":"[redacted]"}`)}},
	}}
	zeroPlan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.secrets.s": {NewState: &structvar.StructVarJSON{Value: []byte(`{"name":"s"}`)}},
	}}
	jobsPlan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.j": {NewState: &structvar.StructVarJSON{Value: []byte(`{"name":"j"}`)}},
	}}

	tests := []struct {
		name    string
		root    *config.Root
		plan    *deployplan.Plan
		wantErr string
	}{
		{
			name: "plan without sensitive resources needs nothing",
			root: secretRoot(t, "", variables),
			plan: jobsPlan,
		},
		{
			name: "sensitive value set in the config passes",
			root: secretRoot(t, secretLiteral, variables),
			plan: secretPlan,
		},
		{
			name: "zero in the plan and set in the config passes",
			root: secretRoot(t, secretLiteral, variables),
			plan: zeroPlan,
		},
		{
			name:    "empty config value is reported",
			root:    secretRoot(t, "", variables),
			plan:    secretPlan,
			wantErr: msgValueMissing,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := CheckSensitivePlanValues(tc.root, tc.plan)
			if tc.wantErr == "" {
				assert.Empty(t, errs)
				return
			}
			require.Len(t, errs, 1)
			assert.EqualError(t, errs[0], tc.wantErr)
		})
	}
}

// TestCheckSensitivePlanValuesOneErrorPerResource checks that each resource with a missing value gets its
// own error, and that a resource missing from the config is reported once.
func TestCheckSensitivePlanValuesOneErrorPerResource(t *testing.T) {
	root := syncedRoot(t, &config.Root{Resources: config.Resources{
		Secrets: map[string]*resources.Secret{
			"a": {Secret: catalog.Secret{Name: "a"}},
			"b": {Secret: catalog.Secret{Name: "b"}},
		},
	}})
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.secrets.a":    {NewState: &structvar.StructVarJSON{Value: []byte(`{"name":"a","value":"[redacted]"}`)}},
		"resources.secrets.b":    {NewState: &structvar.StructVarJSON{Value: []byte(`{"name":"b","value":"[redacted]"}`)}},
		"resources.secrets.gone": {NewState: &structvar.StructVarJSON{Value: []byte(`{"name":"gone","value":"[redacted]"}`)}},
	}}

	errs := CheckSensitivePlanValues(root, plan)
	require.Len(t, errs, 3)
	assert.EqualError(t, errs[0], "resource resources.secrets.a: the config has no value for sensitive field value; set the variable it uses with --var or BUNDLE_VAR_<name>")
	assert.EqualError(t, errs[1], "resource resources.secrets.b: the config has no value for sensitive field value; set the variable it uses with --var or BUNDLE_VAR_<name>")
	assert.EqualError(t, errs[2], msgNotInConfig)
}

// TestResolveReferencesAfterApplyKeepsRedactedEntry checks that apply resolving a reference does not write
// the resolved value back into the plan entry, which would put the secret back into memory after redaction.
func TestResolveReferencesAfterApplyKeepsRedactedEntry(t *testing.T) {
	b := &DeploymentBundle{Plan: deployplan.NewPlanDirect()}
	b.Plan.Plan["resources.catalogs.c"] = &deployplan.PlanEntry{Action: deployplan.Skip}
	b.RemoteStateCache.Store("resources.catalogs.c", &catalog.CatalogInfo{Name: "resolved-name"})

	redactedJSON := []byte(`{"name":"s","value":"[redacted]"}`)
	entry := &deployplan.PlanEntry{Action: deployplan.Create, NewState: &structvar.StructVarJSON{Value: redactedJSON}}
	b.StateCache.Store("resources.secrets.s", &structvar.StructVar{
		Value: &catalog.Secret{Name: "s"},
		Refs:  map[string]string{"value": "${resources.catalogs.c.name}"},
	})

	require.True(t, b.resolveReferences(t.Context(), "resources.secrets.s", entry, "cannot create", false))

	sv, ok := b.StateCache.Load("resources.secrets.s")
	require.True(t, ok)
	assert.Equal(t, "resolved-name", sv.Value.(*catalog.Secret).Value)
	assert.JSONEq(t, string(redactedJSON), string(entry.NewState.Value))
}
