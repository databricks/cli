package phases

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestUploadFileSizeBucket(t *testing.T) {
	tests := []struct {
		size     int64
		expected int
	}{
		{size: 0, expected: 0},
		{size: 255, expected: 0},
		// A size equal to a bound belongs in the next bucket (bounds are exclusive).
		{size: 256, expected: 1},
		{size: 511, expected: 1},
		{size: 512, expected: 2},
		{size: 1023, expected: 2},
		{size: 1 << 10, expected: 3},
		{size: 1440, expected: 3},
		{size: 2<<10 - 1, expected: 3},
		{size: 2 << 10, expected: 4},
		{size: 256 << 10, expected: 11},
		{size: 512 << 10, expected: 12},
		{size: 1 << 20, expected: 13},
		{size: 8 << 20, expected: 16},
		{size: 64<<20 - 1, expected: 18},
		// The final bucket captures everything at least as large as the last bound.
		{size: 64 << 20, expected: 19},
		{size: 1 << 40, expected: 19},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, uploadFileSizeBucket(tc.size), "size=%d", tc.size)
	}
}

type fakeSizer int64

func (s fakeSizer) Size() (int64, bool) { return int64(s), true }

// unknownSizer models a file whose size cannot be determined (stat failure).
type unknownSizer struct{}

func (unknownSizer) Size() (int64, bool) { return 0, false }

func TestUploadFileSizeHistogram(t *testing.T) {
	files := []sizer{
		fakeSizer(0),        // bucket 0  (< 256 B)
		fakeSizer(255),      // bucket 0
		fakeSizer(256),      // bucket 1  (256 B is an exclusive bound -> next bucket)
		fakeSizer(1440),     // bucket 3  (1-2 KiB)
		fakeSizer(64 << 20), // bucket 19 (>= 64 MiB, final bucket)
		fakeSizer(1 << 40),  // bucket 19
	}

	expected := make([]int64, len(uploadFileSizeBucketBounds))
	expected[0] = 2
	expected[1] = 1
	expected[3] = 1
	expected[19] = 2

	assert.Equal(t, expected, uploadFileSizeHistogram(files))
}

// A single unmeasurable file omits the whole histogram, so we never emit a
// partial (and therefore misleading) distribution.
func TestUploadFileSizeHistogramUnknownSizeOmitsHistogram(t *testing.T) {
	files := []sizer{fakeSizer(100), unknownSizer{}, fakeSizer(2000)}
	assert.Nil(t, uploadFileSizeHistogram(files))
}

func TestUploadFileSizeHistogramEmpty(t *testing.T) {
	assert.Nil(t, uploadFileSizeHistogram(nil))
}

// aiRuntimeJob builds a job with a single ai_runtime_task, plus optionally an extra
// no-op task to make it multi-task.
func aiRuntimeJob(extraTask bool) *resources.Job {
	tasks := []jobs.Task{{
		TaskKey:       "train",
		AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp"},
	}}
	if extraTask {
		tasks = append(tasks, jobs.Task{TaskKey: "prep"})
	}
	return &resources.Job{JobSettings: jobs.JobSettings{Tasks: tasks}}
}

func TestAiRuntimeTaskMetrics(t *testing.T) {
	scheduledJob := aiRuntimeJob(false)
	scheduledJob.Schedule = &jobs.CronSchedule{QuartzCronExpression: "0 0 * * * ?"}

	forEachJob := &resources.Job{JobSettings: jobs.JobSettings{Tasks: []jobs.Task{{
		TaskKey:     "fanout",
		ForEachTask: &jobs.ForEachTask{Task: jobs.Task{AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp"}}},
	}}}}

	tests := []struct {
		name                          string
		jobs                          map[string]*resources.Job
		present, scheduled, multitask bool
	}{
		{
			name: "no jobs",
			jobs: nil,
		},
		{
			name: "job without ai_runtime_task",
			jobs: map[string]*resources.Job{"j": {JobSettings: jobs.JobSettings{Tasks: []jobs.Task{{TaskKey: "notebook"}}}}},
		},
		{
			name:    "single task",
			jobs:    map[string]*resources.Job{"j": aiRuntimeJob(false)},
			present: true,
		},
		{
			name:    "multi-task",
			jobs:    map[string]*resources.Job{"j": aiRuntimeJob(true)},
			present: true, multitask: true,
		},
		{
			name:    "scheduled single task",
			jobs:    map[string]*resources.Job{"j": scheduledJob},
			present: true, scheduled: true,
		},
		{
			name:    "for_each nested ai_runtime_task",
			jobs:    map[string]*resources.Job{"j": forEachJob},
			present: true,
		},
		{
			name:    "nil job is skipped",
			jobs:    map[string]*resources.Job{"j": nil, "k": aiRuntimeJob(false)},
			present: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			present, scheduled, multitask := aiRuntimeTaskMetrics(tc.jobs)
			assert.Equal(t, tc.present, present, "present")
			assert.Equal(t, tc.scheduled, scheduled, "scheduled")
			assert.Equal(t, tc.multitask, multitask, "multitask")
		})
	}
}
