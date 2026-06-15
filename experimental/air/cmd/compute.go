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
	return 0, fmt.Errorf("invalid GPU type %q", string(g))
}

// computeConfig is the `compute` block of the run YAML: which accelerators to
// use and how many.
type computeConfig struct {
	NumAccelerators int    `yaml:"num_accelerators"`
	AcceleratorType string `yaml:"accelerator_type"`
	NodePoolID      string `yaml:"node_pool_id"`
	PoolName        string `yaml:"pool_name"`
}

// validate checks the compute block against the backend's constraints.
func (c computeConfig) validate() error {
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

	if c.NodePoolID != "" && c.PoolName != "" {
		return errors.New("compute: cannot specify both node_pool_id and pool_name")
	}

	return nil
}
