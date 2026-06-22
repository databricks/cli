package aircmd

import (
	"fmt"
	"path"
	"strconv"
	"strings"
)

// supportedFilterKeys are the keys accepted by `air list --filter KEY=VALUE`.
var supportedFilterKeys = []string{"accelerator_type", "experiment", "num_accelerators", "user"}

// listFilters holds the parsed `--filter` values for `air list`.
type listFilters struct {
	// User is an exact creator-email match
	User string
	// Experiment is a case-insensitive glob
	Experiment string
	// AcceleratorType is a case-insensitive substring matched against the
	// display GPU name (e.g. "H100").
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
		switch key {
		case "user":
			f.User = value
		case "experiment":
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

// matches reports whether a workflow satisfies the experiment, accelerator-type
// and accelerator-count filters. These have no ListTrainingWorkflows equivalent,
// so they are applied client-side to the response. The user filter is handled by
// the server (via creator_name), so it is not re-checked here.
func (f listFilters) matches(w *trainingWorkflow) bool {
	if f.Experiment != "" {
		name := stripExperimentUserPrefix(w.Status.Mlflow.Experiment)
		matched, err := path.Match(strings.ToLower(f.Experiment), strings.ToLower(name))
		if err != nil || !matched {
			return false
		}
	}

	if f.AcceleratorType != "" || f.NumAccelerators != nil {
		compute := w.Spec.Compute
		if f.AcceleratorType != "" {
			display := strings.ToLower(gpuDisplayName(compute.HardwareAcceleratorType))
			if !strings.Contains(display, strings.ToLower(f.AcceleratorType)) {
				return false
			}
		}
		if f.NumAccelerators != nil && compute.AcceleratorCount != *f.NumAccelerators {
			return false
		}
	}

	return true
}
