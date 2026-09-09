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
	gpuType1xA10  gpuType = "GPU_1xA10"
	gpuType8xH100 gpuType = "GPU_8xH100"
	gpuType1xH100 gpuType = "GPU_1xH100"
)

// gpuTypes lists every valid type. Used for validation error messages.
var gpuTypes = []gpuType{gpuType1xA10, gpuType1xH100, gpuType8xH100}

func validGPUTypesHint() string {
	names := make([]string, len(gpuTypes))
	for i, g := range gpuTypes {
		names[i] = string(g)
	}
	return "valid types are: " + strings.Join(names, ", ")
}

// parseGPUType resolves a YAML accelerator_type string to a gpuType. The match is
// exact: the server's lookup is case-sensitive.
func parseGPUType(value string) (gpuType, error) {
	switch gpuType(value) {
	case gpuType1xA10, gpuType8xH100, gpuType1xH100:
		return gpuType(value), nil
	}
	return "", fmt.Errorf("invalid GPU type %q: %s", value, validGPUTypesHint())
}

// priorityClass is the AI Runtime scheduling priority a run is given within its
// reservation. The values match the AiRuntimeTask.PriorityClass proto enum and
// AICM's PriorityClassValidator (the server-side contract).
type priorityClass string

const (
	priorityClassBestEffort priorityClass = "BEST_EFFORT"
	priorityClassNormal     priorityClass = "NORMAL"
	priorityClassCritical   priorityClass = "CRITICAL"
)

// priorityClasses lists every valid value, low to high. Used for validation
// error messages.
var priorityClasses = []priorityClass{priorityClassBestEffort, priorityClassNormal, priorityClassCritical}

func validPriorityClassesHint() string {
	names := make([]string, len(priorityClasses))
	for i, p := range priorityClasses {
		names[i] = string(p)
	}
	return "valid values are: " + strings.Join(names, ", ")
}

// parsePriorityClass normalizes a YAML priority_class to the enum's contract
// value. Unlike accelerator_type, the match is case-insensitive: the input is
// trimmed and upper-cased, so `critical` and `CRITICAL` both resolve.
func parsePriorityClass(value string) (priorityClass, error) {
	switch p := priorityClass(strings.ToUpper(strings.TrimSpace(value))); p {
	case priorityClassBestEffort, priorityClassNormal, priorityClassCritical:
		return p, nil
	}
	return "", fmt.Errorf("invalid priority_class %q: %s", value, validPriorityClassesHint())
}

// gpusPerNode returns the per-node GPU count, which is the partition count from
// the name (GPU_1xH100 -> 1, GPU_8xH100 -> 8). num_accelerators must be a
// round multiple of this since accelerators are allocated in whole nodes.
func gpusPerNode(g gpuType) (int, error) {
	switch g {
	case gpuType1xA10, gpuType1xH100:
		return 1, nil
	case gpuType8xH100:
		return 8, nil
	}
	// Unreachable: callers resolve g through parseGPUType first, which rejects
	// unknown types. Kept as a defensive guard.
	return 0, fmt.Errorf("invalid GPU type %q", string(g))
}

// computeConfig is the `compute` block of the run YAML: which accelerators to
// use and how many.
type computeConfig struct {
	NumAccelerators       int     `yaml:"num_accelerators" help:"Total number of GPUs to allocate. Must be a positive multiple of the accelerator type's per-node GPU count. See https://docs.databricks.com/aws/en/machine-learning/ai-runtime/cli/yaml-config#reference for supported GPU types."`
	AcceleratorType       string  `yaml:"accelerator_type" help:"Which accelerator to run on, e.g. GPU_1xA10. See https://docs.databricks.com/aws/en/machine-learning/ai-runtime/cli/yaml-config#reference for the current list of supported GPU types. Matched case-sensitively."`
	ProvisionedCapacityID *string `yaml:"provisioned_capacity_id" help:"Pre-provisioned AIR capacity reservation id. Must be 1-255 characters. Contact your Databricks account team to provision capacity."`
	PriorityClass         *string `yaml:"priority_class" help:"Scheduling priority within the reservation: BEST_EFFORT (lowest, preemptable), NORMAL, or CRITICAL (highest). Requires provisioned_capacity_id."`
}

// validate checks the compute block against the backend's constraints.
func (c *computeConfig) validate() error {
	g, err := parseGPUType(c.AcceleratorType)
	if err != nil {
		return fmt.Errorf("compute.accelerator_type: %w", err)
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

	if c.ProvisionedCapacityID != nil {
		v := strings.TrimSpace(*c.ProvisionedCapacityID)
		if v == "" {
			return errors.New("compute.provisioned_capacity_id cannot be empty")
		}
		if len(v) > 255 {
			return fmt.Errorf("compute.provisioned_capacity_id must be 255 characters or less, got %d", len(v))
		}
		*c.ProvisionedCapacityID = v
	}

	if c.PriorityClass != nil {
		p, err := parsePriorityClass(*c.PriorityClass)
		if err != nil {
			return fmt.Errorf("compute.priority_class: %w", err)
		}
		// A priority class only ranks pending work within a reservation, so it
		// requires one.
		if c.ProvisionedCapacityID == nil {
			return errors.New("compute.priority_class requires compute.provisioned_capacity_id — priority applies only to a pre-provisioned capacity reservation")
		}
		*c.PriorityClass = string(p)
	}

	return nil
}
