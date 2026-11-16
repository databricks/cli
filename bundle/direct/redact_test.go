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
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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
	msgValueMissing      = "resource resources.secrets.s: the config has no value for sensitive field value; pass the value"
	msgNotInConfig       = "resource resources.secrets.gone is not in the config; run 'bundle plan' again"
	msgCredentialMissing = "sensitive variable api_credential has no value; pass it with --var api_credential=... or BUNDLE_VAR_api_credential when deploying from a saved plan (the variable-overrides file is not read for sensitive variables)"
	msgVarsMissing       = "sensitive variables api_credential, api_header have no value; pass each with --var NAME=... or BUNDLE_VAR_NAME when deploying from a saved plan (the variable-overrides file is not read for sensitive variables)"
)

func TestRestoreSensitiveFields(t *testing.T) {
	adapter := secretAdapter(t)

	tests := []struct {
		name        string
		resourceKey string
		state       *catalog.Secret
		configValue string
		wantValue   string
		wantErr     string
	}{
		{
			name:        "redaction marker is replaced by the config value",
			resourceKey: "resources.secrets.s",
			state:       &catalog.Secret{Name: "s", Value: sensitiveRedactedValue},
			configValue: secretLiteral,
			wantValue:   secretLiteral,
		},
		{
			name:        "plaintext from a plan saved by an older CLI is replaced by the config value",
			resourceKey: "resources.secrets.s",
			state:       &catalog.Secret{Name: "s", Value: "stale-plaintext"},
			configValue: secretLiteral,
			wantValue:   secretLiteral,
		},
		{
			name:        "zero in the plan but set in the config is restored",
			resourceKey: "resources.secrets.s",
			state:       &catalog.Secret{Name: "s"},
			configValue: secretLiteral,
			wantValue:   secretLiteral,
		},
		{
			name:        "zero in both is skipped",
			resourceKey: "resources.secrets.s",
			state:       &catalog.Secret{Name: "s"},
			configValue: "",
			wantValue:   "",
		},
		{
			name:        "non-zero in the plan with an empty config value is an error",
			resourceKey: "resources.secrets.s",
			state:       &catalog.Secret{Name: "s", Value: sensitiveRedactedValue},
			configValue: "",
			wantErr:     msgValueMissing,
		},
		{
			name:        "non-zero in the plan with the resource missing from config is an error",
			resourceKey: "resources.secrets.gone",
			state:       &catalog.Secret{Name: "s", Value: sensitiveRedactedValue},
			configValue: secretLiteral,
			wantErr:     msgNotInConfig,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := secretRoot(t, tc.configValue, nil)
			err := restoreSensitiveFields(adapter, tc.resourceKey, tc.state, root)
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantValue, tc.state.Value)
			assert.Empty(t, tc.state.EffectiveValue)
		})
	}
}

// TestSensitiveConfigValueNestedField checks that a dotted sensitive field path is read from the
// resource config, not only a top-level field.
func TestSensitiveConfigValueNestedField(t *testing.T) {
	root := syncedRoot(t, &config.Root{Resources: config.Resources{Jobs: map[string]*resources.Job{
		"j": {JobSettings: jobs.JobSettings{Schedule: &jobs.CronSchedule{QuartzCronExpression: "0 0 * * *"}}},
	}}})
	path, err := structpath.ParsePath("schedule.quartz_cron_expression")
	require.NoError(t, err)

	value, err := sensitiveConfigValue(root, "resources.jobs.j", path, "planned")
	require.NoError(t, err)
	assert.Equal(t, "0 0 * * *", value)
}

func TestCheckSensitivePlanValues(t *testing.T) {
	sensitive := true
	variables := func() map[string]*variable.Variable {
		return map[string]*variable.Variable{
			"api_credential": {Sensitive: &sensitive},
			"api_header":     {Sensitive: &sensitive},
			"target_schema":  {},
		}
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
		flags   []string
		env     map[string]string
		wantErr string
	}{
		{
			name: "plan without sensitive resources needs nothing",
			root: secretRoot(t, secretLiteral, variables()),
			plan: jobsPlan,
		},
		{
			name:    "missing variables are named",
			root:    secretRoot(t, secretLiteral, variables()),
			plan:    secretPlan,
			wantErr: msgVarsMissing,
		},
		{
			name:    "empty flag counts as missing even when the environment is set",
			root:    secretRoot(t, secretLiteral, variables()),
			plan:    secretPlan,
			flags:   []string{"api_credential="},
			env:     map[string]string{"BUNDLE_VAR_api_credential": "y", "BUNDLE_VAR_api_header": "y"},
			wantErr: msgCredentialMissing,
		},
		{
			name:  "environment supplies both variables",
			root:  secretRoot(t, secretLiteral, variables()),
			plan:  secretPlan,
			flags: []string{"api_credential=x"},
			env:   map[string]string{"BUNDLE_VAR_api_header": "y"},
		},
		{
			name:  "zero in the plan and set in the config passes",
			root:  secretRoot(t, secretLiteral, variables()),
			plan:  zeroPlan,
			flags: []string{"api_credential=x", "api_header=y"},
		},
		{
			name:    "empty config value",
			root:    secretRoot(t, "", variables()),
			plan:    secretPlan,
			flags:   []string{"api_credential=x", "api_header=y"},
			wantErr: msgValueMissing,
		},
		{
			name:    "resource problem comes before the variable problem",
			root:    secretRoot(t, "", variables()),
			plan:    secretPlan,
			wantErr: msgValueMissing + "; " + msgVarsMissing,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			for k, v := range tc.env {
				ctx = env.Set(ctx, k, v) //nolint:fatcontext // the variables accumulate on one context by design
			}
			err := CheckSensitivePlanValues(ctx, tc.root, tc.plan, tc.flags)
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tc.wantErr)
		})
	}
}
