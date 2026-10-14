package testserver

import (
	"encoding/json"
	clusterpolicies "github.com/databricks/cli/libs/sdktypes/clusterpolicies/v2"
	jobs "github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// A "fixed" element both supplies and enforces its value; a "defaultValue" element only
// supplies one, and only when the request sets apply_policy_default_values.
const testPolicyDefinition = `{
	"spark_version": {"type": "fixed", "value": "policy-version"},
	"custom_tags.Fixed": {"type": "fixed", "value": "f"},
	"custom_tags.Default": {"type": "unlimited", "defaultValue": "d"},
	"node_type_id": {"type": "unlimited", "defaultValue": "policy-node"}
}`

func policyWorkspace() *FakeWorkspace {
	return &FakeWorkspace{ClusterPolicies: map[string]clusterpolicies.Policy{
		"p1": {PolicyId: utils.NonZero("p1"), Definition: utils.NonZero(testPolicyDefinition)},
	}}
}

func TestApplyClusterPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policyID string
		spec     jobs.ClusterSpec_NewCluster
		want     string
		wantErr  string
	}{
		{
			name:     "fixed applies without apply_policy_default_values, defaultValue does not",
			policyID: "p1",
			spec:     jobs.ClusterSpec_NewCluster{PolicyId: utils.NonZero("p1")},
			want:     `{"custom_tags":{"Fixed":"f"},"policy_id":"p1","spark_version":"policy-version"}`,
		},
		{
			name:     "defaultValue applies with apply_policy_default_values",
			policyID: "p1",
			spec:     jobs.ClusterSpec_NewCluster{PolicyId: utils.NonZero("p1"), ApplyPolicyDefaultValues: utils.NonZero(true)},
			want:     `{"apply_policy_default_values":true,"custom_tags":{"Default":"d","Fixed":"f"},"node_type_id":"policy-node","policy_id":"p1","spark_version":"policy-version"}`,
		},
		{
			name:     "a defaultValue does not override what the request supplied",
			policyID: "p1",
			spec:     jobs.ClusterSpec_NewCluster{PolicyId: utils.NonZero("p1"), ApplyPolicyDefaultValues: utils.NonZero(true), NodeTypeId: utils.NonZero("user-node")},
			want:     `{"apply_policy_default_values":true,"custom_tags":{"Default":"d","Fixed":"f"},"node_type_id":"user-node","policy_id":"p1","spark_version":"policy-version"}`,
		},
		{
			name:     "policy tags merge into tags the request supplied",
			policyID: "p1",
			spec:     jobs.ClusterSpec_NewCluster{PolicyId: utils.NonZero("p1"), CustomTags: map[string]string{"Mine": "yes"}},
			want:     `{"custom_tags":{"Fixed":"f","Mine":"yes"},"policy_id":"p1","spark_version":"policy-version"}`,
		},
		{
			name:     "a value contradicting a fixed element is rejected",
			policyID: "p1",
			spec:     jobs.ClusterSpec_NewCluster{PolicyId: utils.NonZero("p1"), SparkVersion: utils.NonZero("user-version")},
			wantErr:  `Cluster validation error: Validation failed for spark_version must be policy-version (is "user-version")`,
		},
		{
			name:     "a tag contradicting a fixed element is rejected",
			policyID: "p1",
			spec:     jobs.ClusterSpec_NewCluster{PolicyId: utils.NonZero("p1"), CustomTags: map[string]string{"Fixed": "mine"}},
			wantErr:  `Cluster validation error: Validation failed for custom_tags, Fixed must be f (is "mine")`,
		},
		{
			name:     "no policy attached leaves the spec untouched",
			policyID: "",
			spec:     jobs.ClusterSpec_NewCluster{SparkVersion: utils.NonZero("v")},
			want:     `{"spark_version":"v"}`,
		},
		{
			name:     "unknown policy leaves the spec untouched",
			policyID: "missing",
			spec:     jobs.ClusterSpec_NewCluster{PolicyId: utils.NonZero("missing"), SparkVersion: utils.NonZero("v")},
			want:     `{"policy_id":"missing","spark_version":"v"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := tt.spec
			msg := policyWorkspace().applyClusterPolicy(&spec, tt.policyID, utils.Deref(spec.ApplyPolicyDefaultValues))

			if tt.wantErr != "" {
				assert.Equal(t, tt.wantErr, msg)
				return
			}
			require.Empty(t, msg)

			got, err := json.Marshal(&spec)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
		})
	}
}
