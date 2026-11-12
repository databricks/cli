package aircmd

import (
	"fmt"
	"path"
	"slices"
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
	// AcceleratorType is the user-provided family, display name, or full type.
	AcceleratorType string
	// AcceleratorTypes are the workspace accelerator types resolved from
	// AcceleratorType. A run matches when its type is in this list.
	AcceleratorTypes []string
	// AcceleratorCount is set when AcceleratorType uses the list table's display
	// format (for example, "8x H100"). It is the run's total accelerator count.
	AcceleratorCount *int
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

// resolveAcceleratorFilter resolves a family, display name, or full type to the
// canonical accelerator types currently available in the workspace.
func resolveAcceleratorFilter(f *listFilters, options []computeOption) error {
	if f.AcceleratorType == "" {
		return nil
	}

	// A full type selects one topology exactly.
	for _, option := range options {
		if strings.EqualFold(f.AcceleratorType, option.HardwareAccelerator) {
			f.AcceleratorTypes = append(f.AcceleratorTypes, option.HardwareAccelerator)
			return nil
		}
	}

	// A copied table value selects a family plus the displayed total count.
	family := f.AcceleratorType
	if count, displayFamily, ok := parseAcceleratorDisplay(f.AcceleratorType); ok {
		f.AcceleratorCount = &count
		family = displayFamily
	}
	for _, option := range options {
		if strings.EqualFold(family, acceleratorFamily(option.HardwareAccelerator)) {
			f.AcceleratorTypes = append(f.AcceleratorTypes, option.HardwareAccelerator)
		}
	}
	if len(f.AcceleratorTypes) == 0 {
		return fmt.Errorf("invalid --filter accelerator_type=%q: %s", f.AcceleratorType, validAcceleratorFilterHint(options))
	}
	return nil
}

// acceleratorFamily extracts the hardware family from a canonical type, so a
// future type such as GPU_4xA100 works without a client-side type list.
func acceleratorFamily(acceleratorType string) string {
	prefix, family, ok := strings.Cut(strings.TrimPrefix(acceleratorType, "GPU_"), "x")
	if ok {
		if _, err := strconv.Atoi(prefix); err == nil && family != "" {
			return family
		}
	}
	return acceleratorType
}

func parseAcceleratorDisplay(value string) (int, string, bool) {
	compact := strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	countText, family, ok := strings.Cut(compact, "x")
	if !ok || family == "" {
		return 0, "", false
	}
	count, err := strconv.Atoi(countText)
	if err != nil || count <= 0 {
		return 0, "", false
	}
	return count, family, true
}

func validAcceleratorFilterHint(options []computeOption) string {
	var families, displays, fullTypes []string
	for _, option := range options {
		family := acceleratorFamily(option.HardwareAccelerator)
		if !slices.ContainsFunc(families, func(v string) bool { return strings.EqualFold(v, family) }) {
			families = append(families, family)
		}
		if option.DisplayName != "" && !slices.ContainsFunc(displays, func(v string) bool { return strings.EqualFold(v, option.DisplayName) }) {
			displays = append(displays, option.DisplayName)
		}
		fullTypes = append(fullTypes, option.HardwareAccelerator)
	}
	if len(fullTypes) == 0 {
		return "no accelerator types are available in this workspace"
	}
	parts := []string{"families (" + strings.Join(families, ", ") + ")"}
	if len(displays) > 0 {
		parts = append(parts, "display names ("+strings.Join(displays, ", ")+")")
	}
	parts = append(parts, "full types ("+strings.Join(fullTypes, ", ")+")")
	return "valid values for this workspace include " + strings.Join(parts, ", ")
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
		if !slices.ContainsFunc(f.AcceleratorTypes, func(v string) bool { return strings.EqualFold(v, fields.GPUType) }) {
			return false
		}
		if f.AcceleratorCount != nil && fields.GPUCount != *f.AcceleratorCount {
			return false
		}
	}
	if f.NumAccelerators != nil && fields.GPUCount != *f.NumAccelerators {
		return false
	}

	return true
}
