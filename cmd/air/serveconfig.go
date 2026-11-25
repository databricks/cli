package aircmd

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
)

// This file defines the `air serve` YAML schema and its validation. Only the model
// and the endpoint are required: what compute is used, and its defaults, follow from
// what the version was registered for, which air serve reads from the version.

// Compute modes, as compute.mode names them.
const (
	modeLatencySensitive    = "latency_sensitive"
	modeThroughputOptimized = "throughput_optimized"
)

// computeModes lists the modes in the order errors name them.
var computeModes = []string{modeLatencySensitive, modeThroughputOptimized}

// modeProfiles are the dedicated compute profiles the modes select.
var modeProfiles = map[string]string{
	modeLatencySensitive:    "COMPUTE_PROFILE_LOW_LATENCY",
	modeThroughputOptimized: "COMPUTE_PROFILE_THROUGHPUT",
}

// acceleratorCPU is the accelerator_type of a replica without a GPU, which a command
// isn't served on: its environment is registered for a GPU.
const acceleratorCPU = "CPU"

// Compute a version gets when the file leaves it out.
const (
	defaultMode = modeLatencySensitive
	// defaultCommandAccelerator serves a version registered with a command, and CPU
	// any other version that isn't a dedicated checkpoint.
	defaultCommandAccelerator = "GPU_1xA10"
	defaultReplicas           = 1
	defaultModelUnits         = 600
)

// servingShape is a replica of serving compute the server offers: what it runs on,
// and the workload type that serves it. fixedScale is set when the endpoint can't
// scale: the server refuses scale to zero and a range of replicas on H100.
type servingShape struct {
	accelerator  string
	workloadType string
	fixedScale   bool
}

// servingShapes lists the replicas a version can be served on, in the order errors
// name them.
var servingShapes = []servingShape{
	{"GPU_1xA10", "GPU_MEDIUM", false},
	{"GPU_4xA10", "MULTIGPU_MEDIUM", false},
	{"GPU_8xA10", "GPU_MEDIUM_8", false},
	{"GPU_1xH100", "GPU_XLARGE", true},
	{"GPU_8xH100", "GPU_XLARGE_8", true},
	{"GPU_1xRTX6000", "GPU_LARGE_RTX", false},
	{acceleratorCPU, workloadTypeCPU, false},
}

// concurrencyPerReplica is the provisioned concurrency one replica of each workload
// type in servingShapes serves. The endpoint takes its scale as concurrency, and the
// server runs concurrency / concurrencyPerReplica replicas.
const concurrencyPerReplica = 4

// replicaLimit keeps a replica count's concurrency within the API's 32-bit field. The
// server's own limit, 512 concurrency unless a workspace's is raised, is lower.
const replicaLimit = math.MaxInt32 / concurrencyPerReplica

// serveConfig is the `air serve` YAML schema: a version of a Unity Catalog model to
// serve on an endpoint.
type serveConfig struct {
	UCModel      string  `yaml:"uc_model" help:"Unity Catalog model to serve a version of, as <catalog>.<schema>.<model>." required:"yes"`
	Endpoint     string  `yaml:"endpoint_name" help:"Serving endpoint to serve the version on. It is created if it doesn't exist. An endpoint that serves one version of uc_model is updated to serve this version instead, with the compute in this file replacing what it has; air serve doesn't change an endpoint that serves anything else." required:"yes"`
	ModelVersion *string `yaml:"model_version" help:"Version of uc_model to serve (default: its newest READY version)."`
	// TODO: env_variables, with {{secrets/scope/key}} values for secrets, set on the
	// served entity.
	Compute *serveComputeConfig `yaml:"compute" help:"Compute to serve on: replicas in gpu, or model_units for a checkpoint registered without engine, which is served with dedicated inference. Every field defaults from what the version was registered for."`
}

// serveComputeConfig is the `compute` block.
type serveComputeConfig struct {
	Mode       *string                `yaml:"mode" help:"For a checkpoint registered without engine, which is served with dedicated inference: latency_sensitive (the default) selects a compute profile tuned for time to first token, and throughput_optimized one tuned for throughput. Case-insensitive. Replicas have no mode yet, so it can't be set with gpu."`
	GPU        *serveGPUConfig        `yaml:"gpu" help:"Replicas to serve on. Not for a checkpoint registered without engine."`
	ModelUnits *serveModelUnitsConfig `yaml:"model_units" help:"Dedicated compute for a checkpoint registered without engine."`
}

// serveGPUConfig is the `compute.gpu` block.
type serveGPUConfig struct {
	AcceleratorType *string      `yaml:"accelerator_type" help:"What each replica runs on: GPU_1xA10, GPU_4xA10, GPU_8xA10, GPU_1xH100, GPU_8xH100, GPU_1xRTX6000, or CPU for a version not registered with a command (default: GPU_1xA10 for a version registered with a command, CPU for any other)."`
	MinReplicas     *wholeNumber `yaml:"min_replicas" help:"Fewest replicas the endpoint scales down to: 0 or more (default: 1). With 0, the endpoint scales to zero when it isn't queried. On GPU_1xH100 and GPU_8xH100 it must equal max_replicas, because endpoints there don't scale."`
	MaxReplicas     *wholeNumber `yaml:"max_replicas" help:"Most replicas the endpoint scales up to: at least 1, and at least min_replicas (default: min_replicas, or 1). Model Serving allows 128 unless your workspace's limit was raised."`
}

// serveModelUnitsConfig is the `compute.model_units` block.
type serveModelUnitsConfig struct {
	MinUnits        *wholeNumber `yaml:"min_units" help:"Model units to provision, above 0 (default: 600). Dedicated endpoints don't scale yet, so min_units and max_units must be equal when both are set."`
	MaxUnits        *wholeNumber `yaml:"max_units" help:"Model units to scale up to. Dedicated endpoints don't scale yet, so it must equal min_units when both are set."`
	ReservationTerm *string      `yaml:"reservation_term" help:"Commitment term of the provisioned units, such as ONE_MONTH. Not supported yet: dedicated endpoints don't accept a commitment term, so a file that sets it is refused before anything changes."`
}

// validate checks the whole config without the version, returning the first
// failure. Fields are checked in declaration order to keep errors stable.
func (c *serveConfig) validate() error {
	if err := validateUCModelName(c.UCModel, "uc_model"); err != nil {
		return err
	}
	if !endpointNameRe().MatchString(c.Endpoint) {
		return fmt.Errorf("endpoint_name must be a serving endpoint name of letters, digits, '-' and '_', got %q", c.Endpoint)
	}
	if c.ModelVersion != nil && !modelVersionRe().MatchString(*c.ModelVersion) {
		return fmt.Errorf("model_version must be a model version number such as 3, got %q", *c.ModelVersion)
	}
	if c.Compute == nil {
		return nil
	}
	return c.Compute.validate()
}

func (c *serveComputeConfig) validate() error {
	if _, err := c.mode(); err != nil {
		return err
	}
	if c.GPU != nil && c.ModelUnits != nil {
		return errors.New("compute can set gpu or model_units, not both: model_units is for a checkpoint registered without engine, and gpu for any other version")
	}
	if c.GPU != nil && c.Mode != nil {
		return errors.New("compute.mode can't be set with compute.gpu: it selects the profile of dedicated inference, and replicas have no mode yet")
	}
	if c.GPU != nil {
		if _, err := c.GPU.shape(defaultCommandAccelerator); err != nil {
			return err
		}
	}
	if c.ModelUnits != nil {
		_, err := c.ModelUnits.units()
		return err
	}
	return nil
}

// mode is compute.mode, lowercase, or the default mode when it isn't set.
func (c *serveComputeConfig) mode() (string, error) {
	if c == nil || c.Mode == nil {
		return defaultMode, nil
	}
	mode := strings.ToLower(*c.Mode)
	if !slices.Contains(computeModes, mode) {
		return "", fmt.Errorf("compute.mode must be one of %s, got %q", strings.Join(computeModes, ", "), *c.Mode)
	}
	return mode, nil
}

// shape returns the serving compute compute.gpu asks for, with accelerator as the
// accelerator when it sets none. A nil block takes every default.
func (g *serveGPUConfig) shape(accelerator string) (servingCompute, error) {
	var gpu serveGPUConfig
	if g != nil {
		gpu = *g
	}
	if gpu.AcceleratorType != nil {
		accelerator = *gpu.AcceleratorType
	}
	minReplicas := defaultReplicas
	if gpu.MinReplicas != nil {
		minReplicas = int(*gpu.MinReplicas)
	}
	// An unset maximum follows the minimum, so min_replicas: 2 alone serves 2.
	maxReplicas := max(minReplicas, defaultReplicas)
	if gpu.MaxReplicas != nil {
		maxReplicas = int(*gpu.MaxReplicas)
	}
	i := slices.IndexFunc(servingShapes, func(s servingShape) bool { return s.accelerator == accelerator })
	if i < 0 {
		return servingCompute{}, fmt.Errorf("compute.gpu.accelerator_type must be one of %s, got %q", acceleratorNames(), accelerator)
	}
	shape := servingShapes[i]
	switch {
	case minReplicas < 0:
		return servingCompute{}, fmt.Errorf("compute.gpu.min_replicas must be 0 or more, got %d", minReplicas)
	case minReplicas > replicaLimit:
		return servingCompute{}, fmt.Errorf("compute.gpu.min_replicas must be at most %d, got %d", replicaLimit, minReplicas)
	case maxReplicas < 1:
		return servingCompute{}, fmt.Errorf("compute.gpu.max_replicas must be at least 1, got %d", maxReplicas)
	case maxReplicas > replicaLimit:
		return servingCompute{}, fmt.Errorf("compute.gpu.max_replicas must be at most %d, got %d", replicaLimit, maxReplicas)
	case minReplicas > maxReplicas:
		return servingCompute{}, fmt.Errorf("compute.gpu.min_replicas must be at most max_replicas, got %d and %d", minReplicas, maxReplicas)
	case shape.fixedScale && minReplicas != maxReplicas:
		return servingCompute{}, fmt.Errorf("compute.gpu.min_replicas must equal max_replicas on %s, because endpoints there don't scale; got %d and %d", shape.accelerator, minReplicas, maxReplicas)
	}
	return servingCompute{
		WorkloadType:   shape.workloadType,
		MinConcurrency: minReplicas * concurrencyPerReplica,
		MaxConcurrency: maxReplicas * concurrencyPerReplica,
		ScaleToZero:    minReplicas == 0,
	}, nil
}

// acceleratorNames lists the accelerator types serving compute supports.
func acceleratorNames() string {
	names := make([]string, 0, len(servingShapes))
	for _, shape := range servingShapes {
		names = append(names, shape.accelerator)
	}
	return strings.Join(names, ", ")
}

// units is the model units compute.model_units asks for. A nil block takes the
// default. Dedicated endpoints take a single size, so a range is refused.
func (m *serveModelUnitsConfig) units() (int64, error) {
	if m == nil {
		return defaultModelUnits, nil
	}
	// TODO: a range and a commitment term, once dedicated endpoints accept them.
	if m.ReservationTerm != nil {
		return 0, errors.New("compute.model_units.reservation_term isn't supported yet: dedicated endpoints don't accept a commitment term; leave it out")
	}
	units := wholeNumber(defaultModelUnits)
	switch {
	case m.MinUnits != nil && m.MaxUnits != nil && *m.MinUnits != *m.MaxUnits:
		return 0, fmt.Errorf("compute.model_units.min_units must equal max_units, because dedicated endpoints don't scale yet; got %d and %d", *m.MinUnits, *m.MaxUnits)
	case m.MinUnits != nil:
		units = *m.MinUnits
	case m.MaxUnits != nil:
		units = *m.MaxUnits
	}
	if units <= 0 {
		return 0, fmt.Errorf("compute.model_units must be above 0, got %d", units)
	}
	return int64(units), nil
}

// servedEntity is the entity the endpoint serves for version, which pkg describes:
// dedicated compute for a dedicated checkpoint, and serving compute for any other
// version. A compute block of the other kind is refused.
func (c *serveConfig) servedEntity(version string, pkg versionPackage) (servedEntity, error) {
	entity := servedEntity{EntityName: c.UCModel, EntityVersion: version}
	var compute serveComputeConfig
	if c.Compute != nil {
		compute = *c.Compute
	}
	described := fmt.Sprintf("version %s of %s", version, c.UCModel)
	if pkg.dedicated {
		if compute.GPU != nil {
			return servedEntity{}, fmt.Errorf("compute.gpu can't be set: %s is a checkpoint registered without engine, which is served with dedicated inference; set compute.model_units, or leave compute out", described)
		}
		mode, err := compute.mode()
		if err != nil {
			return servedEntity{}, err
		}
		units, err := compute.ModelUnits.units()
		if err != nil {
			return servedEntity{}, err
		}
		entity.Dedicated = &dedicatedCompute{ComputeProfile: modeProfiles[mode], ProvisionedModelUnits: units}
		return entity, nil
	}
	if compute.ModelUnits != nil {
		return servedEntity{}, fmt.Errorf("compute.model_units can't be set: it's for a checkpoint registered without engine, and %s isn't one; set compute.gpu, or leave compute out", described)
	}
	if compute.Mode != nil {
		return servedEntity{}, fmt.Errorf("compute.mode can't be set: %s is served on replicas, which have no mode yet; leave it out", described)
	}
	accelerator := acceleratorCPU
	if pkg.command {
		accelerator = defaultCommandAccelerator
		if compute.GPU != nil && compute.GPU.AcceleratorType != nil && *compute.GPU.AcceleratorType == acceleratorCPU {
			return servedEntity{}, fmt.Errorf("compute.gpu.accelerator_type can't be CPU: %s is served by its command, whose environment is registered for a GPU", described)
		}
	}
	serving, err := compute.GPU.shape(accelerator)
	if err != nil {
		return servedEntity{}, err
	}
	entity.Serving = serving
	return entity, nil
}

// serveSchema is the reflection walk over serveConfig, shared by
// `air serve -h config.<field>` and --override path validation.
func serveSchema() configField {
	return configField{
		path: configHelpRoot,
		help: `Serving YAML schema. For details on a field, pass its path to -h, for example:

    databricks air serve -h config.compute

  The newest READY version, on the compute it was registered for:
    uc_model: main.serving.qwen30b_sglang
    endpoint_name: qwen3-30b-sglang

  A version on replicas of your choice:
    uc_model: main.serving.qwen30b_sglang
    endpoint_name: qwen3-30b-sglang
    model_version: 3
    compute:
      gpu:
        accelerator_type: GPU_1xA10
        min_replicas: 0
        max_replicas: 2

  A checkpoint registered without engine, on dedicated inference:
    uc_model: main.serving.gemma4
    endpoint_name: gemma4
    compute:
      mode: throughput_optimized
      model_units:
        min_units: 600
        max_units: 600`,
		children: describeStruct(reflect.TypeFor[serveConfig](), configHelpRoot),
	}
}
