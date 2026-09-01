package deployplan_test

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
)

func TestParentKey(t *testing.T) {
	tests := []struct {
		resourceKey string
		want        string
	}{
		{"resources.schemas.foo.grants", "resources.schemas.foo"},
		{"resources.jobs.foo.permissions", "resources.jobs.foo"},
		{"resources.schemas.foo", ""},
		{"resources.schemas", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.resourceKey, func(t *testing.T) {
			assert.Equal(t, tt.want, deployplan.ParentKey(tt.resourceKey))
		})
	}
}
