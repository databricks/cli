package dresources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/domains"
	"github.com/stretchr/testify/assert"
)

func TestDomainUpdateMask(t *testing.T) {
	always := []string{"subtitle", "description", "icon", "technical_owner_ids", "business_owner_ids"}

	tests := []struct {
		name   string
		config domains.Domain
		want   []string
	}{
		{
			// Writing draft here would publish a domain that is a draft by an account
			// default, or that was set to draft outside of the bundle.
			name:   "draft is left alone when unset",
			config: domains.Domain{TagKey: "Payments"},
			want:   always,
		},
		{
			name:   "draft is written when true",
			config: domains.Domain{TagKey: "Payments", Draft: true},
			want:   append(append([]string{}, always...), "draft"),
		},
		{
			name:   "draft is written when explicitly false",
			config: domains.Domain{TagKey: "Payments", ForceSendFields: []string{"Draft"}},
			want:   append(append([]string{}, always...), "draft"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, domainUpdateMask(&tt.config))
		})
	}
}
