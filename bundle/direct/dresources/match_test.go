package dresources

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindMatchingRule(t *testing.T) {
	rules := []FieldRule{
		{Field: structpath.MustParsePattern("tasks[*].run_if"), Reason: "first"},
		{Field: structpath.MustParsePattern("aws_attributes"), Reason: "second"},
	}

	tests := []struct {
		name       string
		path       string
		wantReason string
		wantOk     bool
	}{
		{
			name:       "wildcard match",
			path:       "tasks[task_key='t1'].run_if",
			wantReason: "first",
			wantOk:     true,
		},
		{
			name:       "prefix match on nested path",
			path:       "aws_attributes.availability",
			wantReason: "second",
			wantOk:     true,
		},
		{
			name:   "no match",
			path:   "description",
			wantOk: false,
		},
		{
			name:   "path shorter than pattern",
			path:   "tasks[task_key='t1']",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)
			reason, ok := FindMatchingRule(path, rules)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}

func TestMatchesAllowedValue(t *testing.T) {
	tests := []struct {
		name   string
		remote any
		values []json.RawMessage
		want   bool
	}{
		{
			name:   "string match",
			remote: "ALL_SUCCESS",
			values: []json.RawMessage{json.RawMessage(`"ALL_SUCCESS"`)},
			want:   true,
		},
		{
			name:   "string no match",
			remote: "ALL_DONE",
			values: []json.RawMessage{json.RawMessage(`"ALL_SUCCESS"`)},
			want:   false,
		},
		{
			name:   "one of several values",
			remote: "SINGLE_TASK",
			values: []json.RawMessage{json.RawMessage(`"MULTI_TASK"`), json.RawMessage(`"SINGLE_TASK"`)},
			want:   true,
		},
		{
			name:   "int match",
			remote: int64(0),
			values: []json.RawMessage{json.RawMessage(`0`)},
			want:   true,
		},
		{
			name:   "bool match",
			remote: false,
			values: []json.RawMessage{json.RawMessage(`false`)},
			want:   true,
		},
		{
			name:   "type mismatch",
			remote: int64(1),
			values: []json.RawMessage{json.RawMessage(`"1"`)},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MatchesAllowedValue(tt.remote, tt.values))
		})
	}
}

func TestMatchesBackendDefault(t *testing.T) {
	cfg := &ResourceLifecycleConfig{
		BackendDefaults: []BackendDefaultRule{
			{Field: structpath.MustParsePattern("run_as")},
			{Field: structpath.MustParsePattern("tasks[*].run_if"), Values: []json.RawMessage{json.RawMessage(`"ALL_SUCCESS"`)}},
		},
	}

	tests := []struct {
		name   string
		cfg    *ResourceLifecycleConfig
		path   string
		remote any
		want   bool
	}{
		{
			name:   "rule without values matches any remote value",
			cfg:    cfg,
			path:   "run_as",
			remote: map[string]any{"user_name": "someone@example.com"},
			want:   true,
		},
		{
			name:   "rule with values matches allowed value",
			cfg:    cfg,
			path:   "tasks[task_key='t1'].run_if",
			remote: "ALL_SUCCESS",
			want:   true,
		},
		{
			name:   "rule with values rejects other value",
			cfg:    cfg,
			path:   "tasks[task_key='t1'].run_if",
			remote: "ALL_DONE",
			want:   false,
		},
		{
			name:   "no rule for path",
			cfg:    cfg,
			path:   "description",
			remote: "x",
			want:   false,
		},
		{
			name:   "nil remote never matches",
			cfg:    cfg,
			path:   "run_as",
			remote: nil,
			want:   false,
		},
		{
			name:   "nil config never matches",
			cfg:    nil,
			path:   "run_as",
			remote: "x",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.cfg.MatchesBackendDefault(path, tt.remote))
		})
	}
}
