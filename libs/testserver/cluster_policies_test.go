package testserver

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
)

const testPolicyDefinition = `{
	"spark_version": {"type": "fixed", "value": "policy-version"},
	"custom_tags.Fixed": {"type": "fixed", "value": "f"},
	"custom_tags.Default": {"type": "unlimited", "defaultValue": "d"},
	"node_type_id": {"type": "forbidden"}
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
			want:          `{"apply_policy_default_values":true,"custom_tags":{"Default":"d","Fixed":"f"},"policy_id":"p1","spark_version":"policy-version"}`,
			wantForceSend: []string{"SparkVersion"},
		},
		{
			name:     "a value the request supplied is not overridden",
			policyID: "p1",
			spec:     compute.ClusterSpec{PolicyId: "p1", SparkVersion: "user-version"},
			want:     `{"custom_tags":{"Fixed":"f"},"policy_id":"p1","spark_version":"user-version"}`,
		},
		{
			name:     "policy tags merge into tags the request supplied",
			policyID: "p1",
			spec:     compute.ClusterSpec{PolicyId: "p1", SparkVersion: "v", CustomTags: map[string]string{"Mine": "yes"}},
			want:     `{"custom_tags":{"Fixed":"f","Mine":"yes"},"policy_id":"p1","spark_version":"v"}`,
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
			policyWorkspace().applyClusterPolicy(&spec, tt.policyID, spec.ApplyPolicyDefaultValues)

			got, err := json.Marshal(&spec)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
			// Only attributes the policy actually supplied may become force-sent. Anything
			// else would serialize as an explicit zero and break the fake's ability to model
			// fields the real API drops (the Jobs API drops apply_policy_default_values).
			assert.Equal(t, tt.wantForceSend, spec.ForceSendFields)
		})
	}
}
