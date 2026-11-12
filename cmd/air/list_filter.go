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
	// AcceleratorType is the user-provided family or full type.
	AcceleratorType string
	// ResolvedAcceleratorType is the canonical-looking user input used for an
	// exact full-type match against runs.
	ResolvedAcceleratorType string
	// AcceleratorFamilyMatch preserves the legacy case-insensitive substring
	// match against the display GPU family.
	AcceleratorFamilyMatch bool
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
		if !slices.Contains(supportedFilterKeys, key) {
			return listFilters{}, fmt.Errorf("unsupported --filter key %q: supported keys are %s", key, strings.Join(supportedFilterKeys, ", "))
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return listFilters{}, fmt.Errorf("invalid --filter %s=%q: value cannot be empty", key, value)
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
		}
	}
	return f, nil
}

// configureAcceleratorFilter selects exact full-type matching for YAML-style
// values and preserves legacy family substring matching for everything else.
func configureAcceleratorFilter(f *listFilters) {
	if strings.HasPrefix(strings.ToUpper(f.AcceleratorType), "GPU_") {
		f.ResolvedAcceleratorType = f.AcceleratorType
		return
	}
	f.AcceleratorFamilyMatch = true
}

// validateAcceleratorFilter checks a filter that produced no runs against the
// accelerator types currently available in the workspace.
func validateAcceleratorFilter(value string, options []computeOption) error {
	for _, option := range options {
		if strings.EqualFold(value, option.HardwareAccelerator) {
			return nil
		}
	}

	if !strings.HasPrefix(strings.ToUpper(value), "GPU_") {
		for _, option := range options {
			family := acceleratorFamily(option.HardwareAccelerator)
			if strings.Contains(strings.ToLower(family), strings.ToLower(value)) {
				return nil
			}
		}
	}
	if fullType := suggestedAcceleratorType(value, options); fullType != "" {
		return fmt.Errorf("invalid --filter accelerator_type=%q: use %q", value, fullType)
	}
	return fmt.Errorf("invalid --filter accelerator_type=%q: %s", value, validAcceleratorFilterHint(options))
}

// acceleratorFamily extracts the hardware family from a canonical type without
// a client-side type list.
func acceleratorFamily(acceleratorType string) string {
	prefix, family, ok := strings.Cut(acceleratorShorthand(acceleratorType), "x")
	if ok {
		if _, err := strconv.Atoi(prefix); err == nil && family != "" {
			return family
		}
	}
	return acceleratorType
}

func acceleratorShorthand(acceleratorType string) string {
	return strings.TrimPrefix(acceleratorType, "GPU_")
}

func acceleratorRunFamily(acceleratorType string) string {
	family := acceleratorFamily(acceleratorType)
	if family != acceleratorType {
		return family
	}
	return gpuDisplayName(acceleratorType)
}

func suggestedAcceleratorType(value string, options []computeOption) string {
	compact := strings.Join(strings.Fields(value), "")
	for _, option := range options {
		if strings.EqualFold(compact, acceleratorShorthand(option.HardwareAccelerator)) {
			return option.HardwareAccelerator
		}
	}
	return ""
}

func validAcceleratorFilterHint(options []computeOption) string {
	var families, fullTypes []string
	for _, option := range options {
		family := acceleratorFamily(option.HardwareAccelerator)
		if !slices.ContainsFunc(families, func(v string) bool { return strings.EqualFold(v, family) }) {
			families = append(families, family)
		}
		fullTypes = append(fullTypes, option.HardwareAccelerator)
	}
	if len(fullTypes) == 0 {
		return "no accelerator types are available in this workspace"
	}
	parts := []string{"families (" + strings.Join(families, ", ") + ")"}
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
		if f.AcceleratorFamilyMatch {
			family := strings.ToLower(acceleratorRunFamily(fields.GPUType))
			if !strings.Contains(family, strings.ToLower(f.AcceleratorType)) {
				return false
			}
		} else if !strings.EqualFold(f.ResolvedAcceleratorType, fields.GPUType) {
			return false
		}
	}
	if f.NumAccelerators != nil && fields.GPUCount != *f.NumAccelerators {
		return false
	}

	return true
}
