package fuzz

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checkJobInvariants asserts the properties that any valid job's create payload
// must satisfy, independent of deploy engine. Unlike a terraform/direct payload
// diff, an invariant has no legitimate reason to fail, so a failure is a real bug
// and the seed reproduces it. Each invariant is checked separately so a failure
// points at the property that broke.
func checkJobInvariants(t require.TestingT, seed int64, job *resources.Job, payload json.RawMessage) {
	p, err := decodePayload(payload)
	require.NoErrorf(t, err, "seed %d: decoding create payload", seed)

	nameMatchesConfig(t, seed, job, p)
	taskKeysMatchConfig(t, seed, job, p)
	dependenciesResolve(t, seed, p)
	jobClusterKeysMatchConfig(t, seed, job, p)
	taskClusterRefsResolve(t, seed, p)
	newClustersSizedExclusively(t, seed, p)
}

// nameMatchesConfig: the engine must not rename the job.
func nameMatchesConfig(t require.TestingT, seed int64, job *resources.Job, p map[string]any) {
	assert.Equalf(t, job.Name, p["name"], "seed %d: payload name must match config", seed)
}

// taskKeysMatchConfig: the payload must carry exactly the tasks from config, no
// more and no fewer, identified by task_key.
func taskKeysMatchConfig(t require.TestingT, seed int64, job *resources.Job, p map[string]any) {
	want := make([]string, 0, len(job.Tasks))
	for _, task := range job.Tasks {
		want = append(want, task.TaskKey)
	}
	assert.ElementsMatchf(t, want, taskKeys(p), "seed %d: payload task keys must match config", seed)
}

// dependenciesResolve: every depends_on must point at a task in the same payload.
func dependenciesResolve(t require.TestingT, seed int64, p map[string]any) {
	keys := sliceToSet(taskKeys(p))
	for _, task := range payloadTasks(p) {
		for _, dep := range slice(task["depends_on"]) {
			d, ok := dep.(map[string]any)
			if !ok {
				continue
			}
			assert.Containsf(t, keys, d["task_key"],
				"seed %d: task %v depends on unknown task %v", seed, task["task_key"], d["task_key"])
		}
	}
}

// jobClusterKeysMatchConfig: the payload's shared job clusters must match config.
func jobClusterKeysMatchConfig(t require.TestingT, seed int64, job *resources.Job, p map[string]any) {
	want := make([]string, 0, len(job.JobClusters))
	for _, jc := range job.JobClusters {
		want = append(want, jc.JobClusterKey)
	}
	assert.ElementsMatchf(t, want, jobClusterKeys(p), "seed %d: payload job cluster keys must match config", seed)
}

// taskClusterRefsResolve: a task referencing a shared cluster must reference one
// declared in job_clusters.
func taskClusterRefsResolve(t require.TestingT, seed int64, p map[string]any) {
	keys := sliceToSet(jobClusterKeys(p))
	for _, task := range payloadTasks(p) {
		ref, ok := task["job_cluster_key"].(string)
		if !ok || ref == "" {
			continue
		}
		assert.Containsf(t, keys, ref,
			"seed %d: task %v references unknown job cluster %q", seed, task["task_key"], ref)
	}
}

// newClustersSizedExclusively: a new_cluster is sized either by autoscale or by a
// fixed num_workers, never both. The two are mutually exclusive cluster shapes, so
// an engine emitting both (e.g. force-sending num_workers onto an autoscale
// cluster) produces a payload the backend rejects.
func newClustersSizedExclusively(t require.TestingT, seed int64, p map[string]any) {
	for _, c := range newClusters(p) {
		_, hasAutoscale := c["autoscale"]
		_, hasNumWorkers := c["num_workers"]
		assert.Falsef(t, hasAutoscale && hasNumWorkers,
			"seed %d: new_cluster must not set both autoscale and num_workers, got %v", seed, c)
	}
}

// decodePayload unmarshals the create body with UseNumber so large int64 values
// (job ids, spark_context_id) aren't corrupted by float64 rounding.
func decodePayload(raw json.RawMessage) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var p map[string]any
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}
	return p, nil
}

// payloadTasks returns the payload's task objects.
func payloadTasks(p map[string]any) []map[string]any {
	tasks := make([]map[string]any, 0, len(slice(p["tasks"])))
	for _, el := range slice(p["tasks"]) {
		if m, ok := el.(map[string]any); ok {
			tasks = append(tasks, m)
		}
	}
	return tasks
}

func taskKeys(p map[string]any) []string {
	var keys []string
	for _, task := range payloadTasks(p) {
		if k, ok := task["task_key"].(string); ok {
			keys = append(keys, k)
		}
	}
	return keys
}

func jobClusterKeys(p map[string]any) []string {
	var keys []string
	for _, el := range slice(p["job_clusters"]) {
		jc, ok := el.(map[string]any)
		if !ok {
			continue
		}
		if k, ok := jc["job_cluster_key"].(string); ok {
			keys = append(keys, k)
		}
	}
	return keys
}

// newClusters returns every new_cluster spec in the payload: one per task that
// defines its own cluster plus one per shared job cluster.
func newClusters(p map[string]any) []map[string]any {
	var specs []map[string]any
	for _, task := range payloadTasks(p) {
		if c, ok := task["new_cluster"].(map[string]any); ok {
			specs = append(specs, c)
		}
	}
	for _, el := range slice(p["job_clusters"]) {
		jc, ok := el.(map[string]any)
		if !ok {
			continue
		}
		if c, ok := jc["new_cluster"].(map[string]any); ok {
			specs = append(specs, c)
		}
	}
	return specs
}

func slice(v any) []any {
	s, _ := v.([]any)
	return s
}

func sliceToSet(s []string) map[string]bool {
	set := make(map[string]bool, len(s))
	for _, v := range s {
		set[v] = true
	}
	return set
}
