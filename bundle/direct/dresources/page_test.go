package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
)

func TestPageUpdateMask(t *testing.T) {
	content := []string{"display_name", "body", "synonyms", "description"}

	tests := []struct {
		name    string
		changes deployplan.Changes
		want    []string
	}{
		{
			name:    "content fields are always written",
			changes: deployplan.Changes{"body": {Action: deployplan.Update}},
			want:    content,
		},
		{
			name:    "state is written when it changes",
			changes: deployplan.Changes{"state": {Action: deployplan.Update}},
			want:    append(append([]string{}, content...), "state"),
		},
		{
			name:    "domain is written when it changes",
			changes: deployplan.Changes{"domain": {Action: deployplan.Update}},
			want:    append(append([]string{}, content...), "domain"),
		},
		{
			name:    "owner_ids is written when it changes",
			changes: deployplan.Changes{"owner_ids": {Action: deployplan.Update}},
			want:    append(append([]string{}, content...), "owner_ids"),
		},
		{
			name: "a change that the plan skips is not written",
			changes: deployplan.Changes{
				"body":      {Action: deployplan.Update},
				"state":     {Action: deployplan.Skip, Reason: "backend_default"},
				"owner_ids": {Action: deployplan.Skip, Reason: "backend_default"},
			},
			want: content,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, pageUpdateMask(&PlanEntry{Changes: tt.changes}))
		})
	}
}

func TestPagePrepareStateTrimsTitle(t *testing.T) {
	input := &resources.Page{DisplayName: "  Chargeback  ", Body: "  body  \n"}

	state := (*ResourcePage)(nil).PrepareState(input)

	assert.Equal(t, "Chargeback", state.DisplayName)
	// The body is Markdown and is stored as is by the API.
	assert.Equal(t, "  body  \n", state.Body)
	// The config is not modified.
	assert.Equal(t, "  Chargeback  ", input.DisplayName)
}
