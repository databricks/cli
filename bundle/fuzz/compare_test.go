package fuzz

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffPayloads(t *testing.T) {
	tests := []struct {
		name      string
		direct    string
		terraform string
		ignore    []string
		want      []string
	}{
		{
			name:      "identical",
			direct:    `{"name":"a","tasks":[{"task_key":"t"}]}`,
			terraform: `{"name":"a","tasks":[{"task_key":"t"}]}`,
			want:      nil,
		},
		{
			name:      "scalar mismatch",
			direct:    `{"name":"a"}`,
			terraform: `{"name":"b"}`,
			want:      []string{"name"},
		},
		{
			name:      "missing on terraform",
			direct:    `{"name":"a","queue":{"enabled":true}}`,
			terraform: `{"name":"a"}`,
			want:      []string{"queue"},
		},
		{
			name:      "missing on direct",
			direct:    `{"name":"a"}`,
			terraform: `{"name":"a","max_concurrent_runs":1}`,
			want:      []string{"max_concurrent_runs"},
		},
		{
			name:      "nested slice element mismatch",
			direct:    `{"tasks":[{"task_key":"t","timeout_seconds":1}]}`,
			terraform: `{"tasks":[{"task_key":"t","timeout_seconds":2}]}`,
			want:      []string{"tasks[0].timeout_seconds"},
		},
		{
			name:      "slice length mismatch",
			direct:    `{"tasks":[{"task_key":"a"},{"task_key":"b"}]}`,
			terraform: `{"tasks":[{"task_key":"a"}]}`,
			want:      []string{"tasks[1]"},
		},
		{
			name:      "number 1 vs 1.0 differ",
			direct:    `{"n":1}`,
			terraform: `{"n":1.0}`,
			want:      []string{"n"},
		},
		{
			name:      "ignored path",
			direct:    `{"tasks":[{"timeout_seconds":1}]}`,
			terraform: `{"tasks":[{"timeout_seconds":2}]}`,
			ignore:    []string{"tasks[*].timeout_seconds"},
			want:      nil,
		},
		{
			name:      "dotted map key is bracket-quoted",
			direct:    `{"spark_conf":{"spark.x.y":"1"}}`,
			terraform: `{"spark_conf":{}}`,
			want:      []string{`spark_conf["spark.x.y"]`},
		},
		{
			name:      "dotted map key can be ignored",
			direct:    `{"c":{"spark_conf":{"spark.x.y":"1"}}}`,
			terraform: `{"c":{"spark_conf":{}}}`,
			ignore:    []string{`c.spark_conf["spark.x.y"]`},
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs, err := DiffPayloads(json.RawMessage(tt.direct), json.RawMessage(tt.terraform), tt.ignore)
			require.NoError(t, err)

			var paths []string
			for _, d := range diffs {
				paths = append(paths, d.Path)
			}
			assert.ElementsMatch(t, tt.want, paths)
		})
	}
}
