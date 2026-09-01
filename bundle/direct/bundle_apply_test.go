package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
)

func TestChildDeletesWithDeletedParent(t *testing.T) {
	tests := []struct {
		name string
		plan map[string]*deployplan.PlanEntry
		want map[string]bool
	}{
		{
			name: "parent deleted too",
			plan: map[string]*deployplan.PlanEntry{
				"resources.schemas.foo":        {Action: deployplan.Delete},
				"resources.schemas.foo.grants": {Action: deployplan.Delete},
			},
			want: map[string]bool{"resources.schemas.foo.grants": true},
		},
		{
			name: "parent stays",
			plan: map[string]*deployplan.PlanEntry{
				"resources.schemas.foo":        {Action: deployplan.Skip},
				"resources.schemas.foo.grants": {Action: deployplan.Delete},
			},
			want: map[string]bool{},
		},
		{
			name: "parent recreated, not deleted",
			plan: map[string]*deployplan.PlanEntry{
				"resources.schemas.foo":        {Action: deployplan.Recreate},
				"resources.schemas.foo.grants": {Action: deployplan.Delete},
			},
			want: map[string]bool{},
		},
		{
			name: "parent not in plan",
			plan: map[string]*deployplan.PlanEntry{
				"resources.schemas.foo.grants": {Action: deployplan.Delete},
			},
			want: map[string]bool{},
		},
		{
			name: "top-level delete is not a child",
			plan: map[string]*deployplan.PlanEntry{
				"resources.schemas.foo": {Action: deployplan.Delete},
			},
			want: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := childDeletesWithDeletedParent(&deployplan.Plan{Plan: tt.plan})
			assert.Equal(t, tt.want, got)
		})
	}
}
