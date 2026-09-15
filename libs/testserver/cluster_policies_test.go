package testserver

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	return &FakeWorkspace{ClusterPolicies: map[string]compute.Policy{
		"p1": {PolicyId: "p1", Definition: testPolicyDefinition},
	}}
}

func TestApplyClusterPolicy(t *testing.T) {
	tests := []struct {
		name          string
		policyID      string
		spec          compute.ClusterSpec
		want          string
		wantForceSend []string
		wantErr       string
	}{
		{
			name:          "fixed applies without apply_policy_default_values, defaultValue does not",
			policyID:      "p1",
			spec:          compute.ClusterSpec{PolicyId: "p1"},
			want:          `{"custom_tags":{"Fixed":"f"},"policy_id":"p1","spark_version":"policy-version"}`,
			wantForceSend: []string{"SparkVersion"},
		},
		{
			name:          "defaultValue applies with apply_policy_default_values",
			policyID:      "p1",
			spec:          compute.ClusterSpec{PolicyId: "p1", ApplyPolicyDefaultValues: true},
			want:          `{"apply_policy_default_values":true,"custom_tags":{"Default":"d","Fixed":"f"},"node_type_id":"policy-node","policy_id":"p1","spark_version":"policy-version"}`,
			wantForceSend: []string{"NodeTypeId", "SparkVersion"},
		},
		{
			name:          "a defaultValue does not override what the request supplied",
			policyID:      "p1",
			spec:          compute.ClusterSpec{PolicyId: "p1", ApplyPolicyDefaultValues: true, NodeTypeId: "user-node"},
			want:          `{"apply_policy_default_values":true,"custom_tags":{"Default":"d","Fixed":"f"},"node_type_id":"user-node","policy_id":"p1","spark_version":"policy-version"}`,
			wantForceSend: []string{"SparkVersion"},
		},
		{
			name:          "policy tags merge into tags the request supplied",
			policyID:      "p1",
			spec:          compute.ClusterSpec{PolicyId: "p1", CustomTags: map[string]string{"Mine": "yes"}},
			want:          `{"custom_tags":{"Fixed":"f","Mine":"yes"},"policy_id":"p1","spark_version":"policy-version"}`,
			wantForceSend: []string{"SparkVersion"},
		},
		{
			name:     "a value contradicting a fixed element is rejected",
			policyID: "p1",
			spec:     compute.ClusterSpec{PolicyId: "p1", SparkVersion: "user-version"},
			wantErr:  `Cluster validation error: Validation failed for spark_version must be policy-version (is "user-version")`,
		},
		{
			name:     "a tag contradicting a fixed element is rejected",
			policyID: "p1",
			spec:     compute.ClusterSpec{PolicyId: "p1", CustomTags: map[string]string{"Fixed": "mine"}},
			wantErr:  `Cluster validation error: Validation failed for custom_tags, Fixed must be f (is "mine")`,
		},
		{
			name:     "no policy attached leaves the spec untouched",
			policyID: "",
			spec:     compute.ClusterSpec{SparkVersion: "v"},
			want:     `{"spark_version":"v"}`,
		},
		{
			name:     "unknown policy leaves the spec untouched",
			policyID: "missing",
			spec:     compute.ClusterSpec{PolicyId: "missing", SparkVersion: "v"},
			want:     `{"policy_id":"missing","spark_version":"v"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := tt.spec
			msg := policyWorkspace().applyClusterPolicy(&spec, tt.policyID, spec.ApplyPolicyDefaultValues)

			if tt.wantErr != "" {
				assert.Equal(t, tt.wantErr, msg)
				return
			}
			require.Empty(t, msg)

			got, err := json.Marshal(&spec)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
			// Only attributes the policy actually supplied may become force-sent. Anything
			// else would serialize as an explicit zero and break the fake's ability to model
			// fields the real API drops (the Jobs API drops apply_policy_default_values).
			assert.Equal(t, tt.wantForceSend, spec.ForceSendFields)
		})
	}
}
