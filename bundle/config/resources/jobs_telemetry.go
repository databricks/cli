package resources

import (
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// Task-type usage is recorded generically at deploy as one "has_<task_type>" key per
// task type present in the bundle (e.g. has_notebook_task, has_ai_runtime_task); see
// collectTaskTypes. The two keys below are extra ai_runtime_task-specific dimensions,
// emitted only when the bundle declares an ai_runtime_task, so a false value means "has
// an ai_runtime_task, but it is not scheduled / not multi-task". GPU type and count are
// intentionally not recorded here.
const (
	aiRuntimeTaskScheduled = "ai_runtime_task_scheduled"
	aiRuntimeTaskMultitask = "ai_runtime_task_multitask"
)

// DeployTelemetryBoolValues records which task types the bundle's jobs use
// (has_<task_type> per type present), plus the ai_runtime_task-specific scheduling and
// multi-task dimensions.
func (*Job) DeployTelemetryBoolValues(group map[string]any) []protos.BoolMapEntry {
	jobs := make(map[string]*Job, len(group))
	for key, r := range group {
		jobs[key] = r.(*Job)
	}

	var values []protos.BoolMapEntry
	for _, key := range collectTaskTypes(jobs) {
		values = append(values, protos.BoolMapEntry{Key: key, Value: true})
	}
	if airPresent, airScheduled, airMultitask := aiRuntimeTaskMetrics(jobs); airPresent {
		values = append(values,
			protos.BoolMapEntry{Key: aiRuntimeTaskScheduled, Value: airScheduled},
			protos.BoolMapEntry{Key: aiRuntimeTaskMultitask, Value: airMultitask},
		)
	}
	return values
}

// addTaskTypeKeys records a "has_<task_type>" key in out for each task-type field
// set on the task. A Jobs task carries its type as one of several optional pointer
// fields (notebook_task, spark_python_task, ai_runtime_task, ...); we reflect over
// the fields whose JSON name ends in "_task" so every task type is captured,
// including ones added later, without changing this code.
func addTaskTypeKeys(task jobs.Task, out map[string]bool) {
	v := reflect.ValueOf(task)
	t := v.Type()
	for i := range t.NumField() {
		field := v.Field(i)
		if field.Kind() != reflect.Pointer || field.IsNil() {
			continue
		}
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if strings.HasSuffix(name, "_task") {
			out["has_"+name] = true
		}
	}
}

// collectTaskTypes returns the sorted "has_<task_type>" telemetry keys for every
// task type used across the bundle's jobs. A for_each_task is unwrapped so the
// wrapped task's type is recorded too. Sorted for deterministic telemetry output.
func collectTaskTypes(jobs map[string]*Job) []string {
	seen := map[string]bool{}
	for _, job := range jobs {
		if job == nil {
			continue
		}
		for _, task := range job.Tasks {
			addTaskTypeKeys(task, seen)
			if task.ForEachTask != nil {
				addTaskTypeKeys(task.ForEachTask.Task, seen)
			}
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// aiRuntimeTaskMetrics computes the ai_runtime_task-specific deploy dimensions.
// present is true when any job declares an ai_runtime_task (including one nested in
// a for_each_task); scheduled and multitask describe those jobs and are meaningful
// only when present is true. Task-type presence itself (has_ai_runtime_task and the
// other has_*_task keys) is recorded generically by collectTaskTypes.
//
// code_source_path is deliberately not inspected: it is rewritten to its uploaded
// remote path before this runs, so it would always read as remote. GPU type and
// count are likewise not recorded here.
func aiRuntimeTaskMetrics(jobs map[string]*Job) (present, scheduled, multitask bool) {
	for _, job := range jobs {
		if job == nil {
			continue
		}
		jobHasAiRuntimeTask := false
		for _, task := range job.Tasks {
			rt := task.AiRuntimeTask
			if rt == nil && task.ForEachTask != nil {
				rt = task.ForEachTask.Task.AiRuntimeTask
			}
			if rt != nil {
				jobHasAiRuntimeTask = true
				break
			}
		}
		if !jobHasAiRuntimeTask {
			continue
		}
		present = true
		if job.Schedule != nil || job.Trigger != nil || job.Continuous != nil {
			scheduled = true
		}
		if len(job.Tasks) > 1 {
			multitask = true
		}
	}
	return present, scheduled, multitask
}
