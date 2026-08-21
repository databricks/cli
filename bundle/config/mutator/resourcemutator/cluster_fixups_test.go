package resourcemutator

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
)

func TestInitializeNumWorkers(t *testing.T) {
	tests := []struct {
		name string
		in   compute.ClusterSpec
		want []string
	}{
		{
			name: "omitted",
			want: []string{"NumWorkers"},
		},
		{
			name: "omitted with policy defaults disabled",
			in: compute.ClusterSpec{
				ForceSendFields: []string{"ApplyPolicyDefaultValues"},
			},
			want: []string{"ApplyPolicyDefaultValues", "NumWorkers"},
		},
		{
			name: "omitted with policy defaults",
			in: compute.ClusterSpec{
				ApplyPolicyDefaultValues: true,
			},
		},
		{
			name: "explicit zero with policy defaults",
			in: compute.ClusterSpec{
				ApplyPolicyDefaultValues: true,
				ForceSendFields:          []string{"NumWorkers"},
			},
			want: []string{"NumWorkers"},
		},
		{
			name: "explicit non-zero with policy defaults",
			in: compute.ClusterSpec{
				ApplyPolicyDefaultValues: true,
				NumWorkers:               2,
			},
		},
		{
			name: "autoscale with policy defaults",
			in: compute.ClusterSpec{
				ApplyPolicyDefaultValues: true,
				Autoscale: &compute.AutoScale{
					MinWorkers: 1,
					MaxWorkers: 4,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := tt.in
			initializeNumWorkers(&tt.in)
			assert.Equal(t, before.NumWorkers, tt.in.NumWorkers)
			assert.Equal(t, before.Autoscale, tt.in.Autoscale)
			assert.Equal(t, tt.want, tt.in.ForceSendFields)
		})
	}
}
