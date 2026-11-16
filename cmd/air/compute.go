package aircmd

import (
	"errors"
	"fmt"
	"strings"
)

// gpuType is a wire-facing accelerator type submitted to the training service.
// The number in the name is the partition count (e.g. GPU_8xH100 is 8 GPUs).
type gpuType string

const (
	gpuType1xA10     gpuType = "GPU_1xA10"
	gpuType8xH100    gpuType = "GPU_8xH100"
	gpuType1xH100    gpuType = "GPU_1xH100"
	gpuType8xB300    gpuType = "GPU_8xB300"
	gpuType1xRTX6000 gpuType = "GPU_1xRTX6000"
)

// parseGPUType resolves a YAML accelerator_type string to a gpuType. The match is
// exact: the server's lookup is case-sensitive. The error lists no types because
// which ones a workspace can use comes from the server, not this CLI.
func parseGPUType(value string) (gpuType, error) {
	switch gpuType(value) {
	case gpuType1xA10, gpuType8xH100, gpuType1xH100, gpuType8xB300, gpuType1xRTX6000:
		return gpuType(value), nil
	}
	return "", fmt.Errorf("%q is not a recognized accelerator type", value)
}

// priorityClass is the AI Runtime scheduling priority a run is given within its
// reservation. The values match the AiRuntimeTask.PriorityClass proto enum and
// AICM's PriorityClassValidator (the server-side contract).
type priorityClass string

const (
	priorityClassBestEffort priorityClass = "BEST_EFFORT"
	priorityClassNormal     priorityClass = "NORMAL"
)

// priorityClasses lists every valid value, low to high. Used for validation
// error messages.
var priorityClasses = []priorityClass{priorityClassBestEffort, priorityClassNormal}

func validPriorityClassesHint() string {
	names := make([]string, len(priorityClasses))
	for i, p := range priorityClasses {
		names[i] = string(p)
	}
	return "valid values are: " + strings.Join(names, ", ")
}

// parsePriorityClass normalizes a YAML priority_class to the enum's contract
// value. Unlike accelerator_type, the match is case-insensitive: the input is
// trimmed and upper-cased, so `normal` and `NORMAL` both resolve.
func parsePriorityClass(value string) (priorityClass, error) {
	switch p := priorityClass(strings.ToUpper(strings.TrimSpace(value))); p {
	case priorityClassBestEffort, priorityClassNormal:
		return p, nil
	}
	return "", fmt.Errorf("invalid priority_class %q: %s", value, validPriorityClassesHint())
}

// gpusPerNode returns the per-node GPU count, which is the partition count from
// the name (GPU_1xH100 -> 1, GPU_8xH100 -> 8). num_accelerators must be a
// round multiple of this since accelerators are allocated in whole nodes.
func gpusPerNode(g gpuType) (int, error) {
	switch g {
	case gpuType1xA10, gpuType1xH100, gpuType1xRTX6000:
		return 1, nil
	case gpuType8xH100, gpuType8xB300:
		return 8, nil
	}
	// Unreachable: callers resolve g through parseGPUType first, which rejects
	// unknown types. Kept as a defensive guard.
	return 0, fmt.Errorf("invalid GPU type %q", string(g))
}

// computeConfig is the `compute` block of the run YAML: which accelerators to
// use and how many.
type computeConfig struct {
	NumAccelerators   int     `yaml:"num_accelerators" help:"Total number of GPUs to allocate. Must be a positive multiple of the accelerator type's per-node GPU count. See Docs below for GPUs per node." required:"yes" docs:"/machine-learning/ai-runtime/connecting/cli/yaml-config#accelerator-types"`
	AcceleratorType   string  `yaml:"accelerator_type" help:"Which accelerator to run on, e.g. GPU_1xA10. Matched case-sensitively." required:"yes" docs:"/machine-learning/ai-runtime/connecting/cli/yaml-config#accelerator-types"`
	AcceleratorPoolID *string `yaml:"accelerator_pool_id" help:"GPU Pool to run on, by ID. Must be 1-255 characters. List available pools with 'databricks air pools list'. An invalid or inaccessible pool is rejected; the workload will not fall back to on-demand capacity. When omitted, workspace capacity selection applies." group:"GPU Pools (optional)"`
	PoolID            *string `yaml:"pool_id" help:"Compatibility alias for accelerator_pool_id. If both are set, their IDs must match." group:"GPU Pools (optional)"`
	PriorityClass     *string `yaml:"priority_class" help:"Scheduling priority within the pool: BEST_EFFORT (lowest, preemptable) or NORMAL. Requires accelerator_pool_id (or its pool_id alias)." group:"GPU Pools (optional)"`
}

// validate checks the compute block against the backend's constraints.
func (c *computeConfig) validate() error {
	g, err := parseGPUType(c.AcceleratorType)
	if err != nil {
		return fmt.Errorf("compute.accelerator_type: %w. Run 'databricks air run -h config.compute' to see the types available in your workspace", err)
	}

	if c.NumAccelerators <= 0 {
		return fmt.Errorf("compute.num_accelerators must be positive, got %d", c.NumAccelerators)
	}

	perNode, err := gpusPerNode(g)
	if err != nil {
		return err
	}
	if c.NumAccelerators%perNode != 0 {
		return fmt.Errorf("compute.num_accelerators for %s must be a multiple of %d, got %d", c.AcceleratorType, perNode, c.NumAccelerators)
	}

	for _, field := range []struct {
		name  string
		value *string
	}{
		{"accelerator_pool_id", c.AcceleratorPoolID},
		{"pool_id", c.PoolID},
	} {
		if field.value == nil {
			continue
		}
		v := strings.TrimSpace(*field.value)
		if v == "" {
			return fmt.Errorf("compute.%s cannot be empty", field.name)
		}
		if len(v) > 255 {
			return fmt.Errorf("compute.%s must be 255 characters or less, got %d", field.name, len(v))
		}
		*field.value = v
	}
	if c.AcceleratorPoolID != nil && c.PoolID != nil && *c.AcceleratorPoolID != *c.PoolID {
		return errors.New("compute.accelerator_pool_id and compute.pool_id must match when both are set")
	}

	if c.PriorityClass != nil {
		p, err := parsePriorityClass(*c.PriorityClass)
		if err != nil {
			return fmt.Errorf("compute.priority_class: %w", err)
		}
		// A priority class only ranks pending work within a pool, so it requires
		// one.
		if c.acceleratorPoolID() == nil {
			return errors.New("compute.priority_class requires compute.accelerator_pool_id (or compute.pool_id) — priority applies only within a GPU pool")
		}
		*c.PriorityClass = string(p)
	}

	return nil
}

// acceleratorPoolID resolves the compatibility alias after config validation.
func (c *computeConfig) acceleratorPoolID() *string {
	if c.AcceleratorPoolID != nil {
		return c.AcceleratorPoolID
	}
	return c.PoolID
}
