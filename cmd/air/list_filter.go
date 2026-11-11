package aircmd

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// supportedFilterKeys are the keys accepted by `air list --filter KEY=VALUE`.
var supportedFilterKeys = []string{"accelerator_type", "experiment", "num_accelerators", "user"}

// hasTaskFilter reports whether any filter is applied to a run's task fields
// (experiment or accelerators), i.e. matched after a run is fetched rather than
// while scanning. The index path uses this to skip its newest-N truncation, so a
// dropped match doesn't shrink the result below --limit.
func (f listFilters) hasTaskFilter() bool {
	return f.Experiment != "" || f.AcceleratorType != "" || f.NumAccelerators != nil
}

// listFilters holds the parsed `--filter` values for `air list`.
type listFilters struct {
	// User is an exact creator-email match
	User string
	// Experiment is a case-insensitive glob
	Experiment string
	// AcceleratorType is a case-insensitive exact match against either the full
	// accelerator type or its display family (e.g. "GPU_8xH100" or "H100").
	AcceleratorType string
	// NumAccelerators is an exact match against the GPU count.
	NumAccelerators *int
}

func parseListFilters(raw []string) (listFilters, error) {
	var f listFilters
	for _, item := range raw {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			return listFilters{}, fmt.Errorf("invalid --filter %q: expected KEY=VALUE", item)
		}
		if strings.TrimSpace(value) == "" {
			return listFilters{}, fmt.Errorf("invalid --filter %s: value cannot be empty", key)
		}
		switch key {
		case "user":
			f.User = value
		case "experiment":
			if _, err := path.Match(value, ""); err != nil {
				return listFilters{}, fmt.Errorf("invalid --filter experiment=%q: invalid glob pattern: %w", value, err)
			}
			f.Experiment = value
		case "accelerator_type":
			if !validAcceleratorFilter(value) {
				return listFilters{}, fmt.Errorf("invalid --filter accelerator_type=%q: %s", value, validAcceleratorFilterHint())
			}
			f.AcceleratorType = value
		case "num_accelerators":
			n, err := strconv.Atoi(value)
			if err != nil || n <= 0 {
				return listFilters{}, fmt.Errorf("invalid --filter num_accelerators=%q: must be a positive integer", value)
			}
			f.NumAccelerators = &n
		default:
			return listFilters{}, fmt.Errorf("unsupported --filter key %q: supported keys are %s", key, strings.Join(supportedFilterKeys, ", "))
		}
	}
	return f, nil
}

// validAcceleratorFilter reports whether value is a full accelerator type or
// a short family name. Full types select one topology; short names match every
// topology in that family (for example, H100 matches both 1x and 8x H100).
func validAcceleratorFilter(value string) bool {
	for _, gpuType := range gpuTypes {
		if strings.EqualFold(value, string(gpuType)) || strings.EqualFold(value, gpuDisplayName(string(gpuType))) {
			return true
		}
	}
	return false
}

func validAcceleratorFilterHint() string {
	return "use a family name (A10, H100, B300, RTX6000) or a full type; " + validGPUTypesHint()
}

// filterFields are the task-filter inputs, cached alongside the row so a cache
// hit can be re-filtered without re-fetching the run.
type filterFields struct {
	Experiment string `json:"experiment"`
	GPUType    string `json:"gpu_type"`
	GPUCount   int    `json:"gpu_count"`
}

func filterFieldsFromRun(run *jobs.Run) filterFields {
	gpuType, count := jobCompute(run)
	return filterFields{
		Experiment: jobExperiment(run),
		GPUType:    gpuType,
		GPUCount:   count,
	}
}

// matches reports whether a run satisfies the experiment, accelerator-type and
// accelerator-count filters. The user filter is applied separately while
// scanning, since it maps onto the run's creator rather than its task.
func (f listFilters) matches(run *jobs.Run) bool {
	return f.matchesFields(filterFieldsFromRun(run))
}

// matchesFields is the shared comparator for live runs and cached rows, so the
// two paths can't drift.
func (f listFilters) matchesFields(fields filterFields) bool {
	if f.Experiment != "" {
		matched, err := path.Match(strings.ToLower(f.Experiment), strings.ToLower(fields.Experiment))
		if err != nil || !matched {
			return false
		}
	}

	if f.AcceleratorType != "" {
		matchesFullType := strings.EqualFold(fields.GPUType, f.AcceleratorType)
		matchesFamily := strings.EqualFold(gpuDisplayName(fields.GPUType), f.AcceleratorType)
		if !matchesFullType && !matchesFamily {
			return false
		}
	}
	if f.NumAccelerators != nil && fields.GPUCount != *f.NumAccelerators {
		return false
	}

	return true
}
