package phases

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/telemetry/protos"
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

func TestRecordResourceBoolValues(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{Resources: config.Resources{
		Jobs: map[string]*resources.Job{
			"nil": nil,
			"a": {JobSettings: jobs.JobSettings{Tasks: []jobs.Task{
				{TaskKey: "train", AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp"}},
				{TaskKey: "nb", NotebookTask: &jobs.NotebookTask{NotebookPath: "/nb"}},
			}}},
		},
	}}}

	recordResourceBoolValues(b)
	assert.Equal(t, []protos.BoolMapEntry{
		{Key: "has_ai_runtime_task", Value: true},
		{Key: "has_notebook_task", Value: true},
		{Key: "ai_runtime_task_scheduled", Value: false},
		{Key: "ai_runtime_task_multitask", Value: true},
	}, b.Metrics.BoolValues)
}

// Every type that supports lifecycle.started declares the key that reports it, only those
// do, and no two declare the same key.
func TestLifecycleStartedTelemetryIsDeclaredByTypesWithStarted(t *testing.T) {
	declaredBy := map[string]string{}
	for _, rt := range config.AllResourceTypes() {
		_, hasStarted := rt.New().GetLifecycle().(resources.LifecycleWithStarted)
		d, declares := rt.New().(resources.LifecycleStartedTelemetry)
		assert.Equal(t, hasStarted, declares, "resource type %q supports lifecycle.started: %v, declares LifecycleStartedTelemetry: %v", rt.Key, hasStarted, declares)
		if !declares {
			continue
		}
		key := d.LifecycleStartedTelemetryKey()
		if other, ok := declaredBy[key]; ok {
			t.Errorf("resource types %q and %q both declare lifecycle.started telemetry key %q", other, rt.Key, key)
			continue
		}
		declaredBy[key] = rt.Key
	}
}

func TestRecordLifecycleStarted(t *testing.T) {
	started := true
	stopped := false
	b := &bundle.Bundle{Config: config.Root{Resources: config.Resources{
		SqlWarehouses: map[string]*resources.SqlWarehouse{"w": {Lifecycle: &resources.LifecycleWithStarted{Started: &stopped}}},
		Clusters:      map[string]*resources.Cluster{"nil": nil, "unset": {}, "c": {Lifecycle: &resources.LifecycleWithStarted{Started: &started}}},
		Apps:          map[string]*resources.App{"a": {Lifecycle: &resources.LifecycleWithStarted{Started: &started}}},
		Jobs:          map[string]*resources.Job{"j": {}},
	}}}

	recordLifecycleStarted(b)
	assert.Equal(t, []protos.BoolMapEntry{
		{Key: "app_lifecycle_started", Value: true},
		{Key: "cluster_lifecycle_started", Value: true},
		{Key: "sql_warehouse_lifecycle_started", Value: false},
	}, b.Metrics.BoolValues)
}

// Every IDs field of the deploy event is filled by exactly one resource type.
func TestDeployTelemetryIDsFieldsAreDeclaredOnce(t *testing.T) {
	var event protos.BundleDeployEvent
	declaredBy := map[*[]string][]string{}
	for key, d := range config.ResourceTypesImplementing[resources.DeployTelemetryIDs]() {
		field := d.DeployTelemetryIDsField(&event)
		declaredBy[field] = append(declaredBy[field], key)
	}

	v := reflect.ValueOf(&event).Elem()
	for i := range v.NumField() {
		name := v.Type().Field(i).Name
		if !strings.HasPrefix(name, "Resource") || !strings.HasSuffix(name, "IDs") {
			continue
		}
		field := v.Field(i).Addr().Interface().(*[]string)
		assert.Len(t, declaredBy[field], 1, "field %s is declared by %v", name, declaredBy[field])
		delete(declaredBy, field)
	}
	assert.Empty(t, declaredBy, "declared fields that are not IDs fields of the deploy event")
}

// Every per-type count field of the deploy event is filled by exactly one resource type.
func TestDeployTelemetryCountFieldsAreDeclaredOnce(t *testing.T) {
	var event protos.BundleDeployEvent
	declaredBy := map[*int64][]string{}
	for key, d := range config.ResourceTypesImplementing[resources.DeployTelemetryCount]() {
		field := d.DeployTelemetryCountField(&event)
		declaredBy[field] = append(declaredBy[field], key)
	}

	v := reflect.ValueOf(&event).Elem()
	for i := range v.NumField() {
		name := v.Type().Field(i).Name
		// ResourceCount counts resources of every type.
		if name == "ResourceCount" || !strings.HasPrefix(name, "Resource") || !strings.HasSuffix(name, "Count") {
			continue
		}
		field := v.Field(i).Addr().Interface().(*int64)
		assert.Len(t, declaredBy[field], 1, "field %s is declared by %v", name, declaredBy[field])
		delete(declaredBy, field)
	}
	assert.Empty(t, declaredBy, "declared fields that are not per-type count fields of the deploy event")
}

func TestRecordResourceCounts(t *testing.T) {
	r := &config.Resources{
		Jobs:    map[string]*resources.Job{"a": {}, "nil": nil},
		Apps:    map[string]*resources.App{"a": {}},
		Secrets: map[string]*resources.Secret{"s": {}},
	}

	var event protos.BundleDeployEvent
	recordResourceCounts(r, &event)
	assert.Equal(t, protos.BundleDeployEvent{ResourceJobCount: 2, ResourceAppCount: 1}, event)
}

func TestRecordResourceIDs(t *testing.T) {
	r := &config.Resources{
		Jobs: map[string]*resources.Job{
			"b":          {ID: "2"},
			"a":          {ID: "1"},
			"nil":        nil,
			"undeployed": {},
		},
		Dashboards: map[string]*resources.Dashboard{"d": {ID: "dash"}},
		Schemas:    map[string]*resources.Schema{"s": {ID: "main.s"}},
	}

	var event protos.BundleDeployEvent
	recordResourceIDs(r, &event)
	assert.Equal(t, []string{"1", "2"}, event.ResourceJobIDs)
	assert.Equal(t, []string{"dash"}, event.ResourceDashboardIDs)
	assert.Nil(t, event.ResourcePipelineIDs)
	assert.Nil(t, event.ResourceClusterIDs)
}

func TestRecordResourceIDsLimit(t *testing.T) {
	jobs := map[string]*resources.Job{}
	for i := range ResourceIdLimit + 1 {
		jobs[strconv.Itoa(i)] = &resources.Job{ID: strconv.Itoa(i)}
	}

	var event protos.BundleDeployEvent
	recordResourceIDs(&config.Resources{Jobs: jobs}, &event)
	assert.Len(t, event.ResourceJobIDs, ResourceIdLimit)
	assert.True(t, slices.IsSorted(event.ResourceJobIDs))
}

func TestPydabsMetrics(t *testing.T) {
	pythonConfig := config.Python{
		Resources: []string{"resources:load_resources"},
		Mutators:  []string{"mutators:m1", "mutators:m2"},
	}
	metrics := bundle.Metrics{
		PythonAddedResources:   map[string]int64{"jobs": 2, "pipelines": 1},
		PythonUpdatedResources: map[string]int64{"jobs": 3, "apps": 1},
		PythonConfigSection:    protos.PydabsConfigSectionPython,
	}
	expected := &protos.BundleDeployPydabs{
		ResourceLoadersCount:  1,
		ResourceMutatorsCount: 2,
		AddedResourcesCount:   3,
		UpdatedResourcesCount: 4,
		Resources: []protos.PydabsResourceTypeCount{
			{ResourceType: "apps", AddedCount: 0, UpdatedCount: 1},
			{ResourceType: "jobs", AddedCount: 2, UpdatedCount: 3},
			{ResourceType: "pipelines", AddedCount: 1, UpdatedCount: 0},
		},
		ConfigSection: protos.PydabsConfigSectionPython,
	}

	b := &bundle.Bundle{Config: config.Root{Experimental: &config.Experimental{Python: pythonConfig}}, Metrics: metrics}
	assert.Equal(t, expected, pydabsMetrics(b))
}
