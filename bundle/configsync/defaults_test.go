package configsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// spark_env_vars is skipped when it is absent from config, both as a whole map and
// per key: a cluster policy can pin a single variable, which produces a per-key
// change path.
func TestShouldSkipFieldSparkEnvVars(t *testing.T) {
	remote := map[string]any{"UV_INDEX_ACME_PASSWORD": "secret"}

	tests := []struct {
		name           string
		path           string
		value          any
		hasConfigValue bool
		want           bool
	}{
		{
			name:  "task cluster whole map",
			path:  "resources.jobs.my_job.tasks[0].new_cluster.spark_env_vars",
			value: remote,
			want:  true,
		},
		{
			name:  "task cluster single key",
			path:  "resources.jobs.my_job.tasks[0].new_cluster.spark_env_vars.UV_INDEX_ACME_PASSWORD",
			value: "secret",
			want:  true,
		},
		{
			name:  "job cluster whole map",
			path:  "resources.jobs.my_job.job_clusters[0].new_cluster.spark_env_vars",
			value: remote,
			want:  true,
		},
		{
			name:  "job cluster single key",
			path:  "resources.jobs.my_job.job_clusters[0].new_cluster.spark_env_vars.UV_INDEX_ACME_PASSWORD",
			value: "secret",
			want:  true,
		},
		{
			name:  "standalone cluster whole map",
			path:  "resources.clusters.my_cluster.spark_env_vars",
			value: remote,
			want:  true,
		},
		{
			name:  "standalone cluster single key",
			path:  "resources.clusters.my_cluster.spark_env_vars.UV_INDEX_ACME_PASSWORD",
			value: "secret",
			want:  true,
		},
		{
			name:           "whole map present in config",
			path:           "resources.jobs.my_job.job_clusters[0].new_cluster.spark_env_vars",
			value:          remote,
			hasConfigValue: true,
			want:           false,
		},
		{
			name:           "single key present in config",
			path:           "resources.jobs.my_job.job_clusters[0].new_cluster.spark_env_vars.UV_INDEX_ACME_PASSWORD",
			value:          "secret",
			hasConfigValue: true,
			want:           false,
		},
		{
			// The scope mirrors the other new_cluster entries (custom_tags,
			// cluster_log_conf), none of which cover for_each_task.
			name:  "for_each_task cluster is not covered",
			path:  "resources.jobs.my_job.tasks[0].for_each_task.task.new_cluster.spark_env_vars",
			value: remote,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldSkipField(tt.path, tt.value, tt.hasConfigValue))
		})
	}
}

func TestIsBackendManagedSkip(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		hasConfigValue bool
		want           bool
	}{
		{
			name: "backend default absent from config",
			path: "resources.jobs.my_job.job_clusters[0].new_cluster.spark_env_vars.UV_INDEX_ACME_PASSWORD",
			want: true,
		},
		{
			name: "backend default map absent from config",
			path: "resources.clusters.my_cluster.custom_tags",
			want: true,
		},
		{
			name:           "backend default present in config",
			path:           "resources.clusters.my_cluster.custom_tags",
			hasConfigValue: true,
			want:           false,
		},
		{
			// The other markers fire on every run, so surfacing them would be noise.
			name: "always skipped field",
			path: "resources.jobs.my_job.edit_mode",
			want: false,
		},
		{
			name: "value-compared default",
			path: "resources.jobs.my_job.timeout_seconds",
			want: false,
		},
		{
			name: "field with no rule",
			path: "resources.jobs.my_job.description",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isBackendManagedSkip(tt.path, tt.hasConfigValue))
		})
	}
}
