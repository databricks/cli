package bundle

// Telemetry mirrors the boolean telemetry flags collected during a deploy so
// they can be walked into the bundle bitmap (see bundle/bitmap). It lives on
// Bundle, not on the input config, so it is invisible to the config schema and
// users cannot set it. Fields have no json tags: the bitmap walker uses the Go
// field names for the schema paths (e.g. "telemetry.SelectUsed").
//
// This is strictly additive: the existing metrics mechanism
// (Metrics.AddBoolValue / SetBoolValue in bundle/metrics) is unchanged and
// still feeds the telemetry wire. Each of its call sites additionally sets the
// matching field here so the same signal reaches the bitmap.
//
// Value-bearing flags whose false is a real measurement (distinct from "not
// measured") are stored as a BoolPair so the presence-only bitmap can represent
// both outcomes.
//
// Not mirrored: the has_<task_type> keys, which the bitmap already records as
// config fields (resources.jobs.*.tasks[*].<type>_task), and the local.cache.*
// keys, which describe the local user cache rather than the bundle.
type Telemetry struct {
	// Single-bit flags: only ever set true when the feature is present, or a
	// one-hot group where the set member already encodes the outcome.
	ArtifactDynamicVersionIsSet   bool
	HasTfOnlyReferences           bool
	SelectUsed                    bool
	ArtifactsReferenceUsed        bool
	ConfigHasDoubleDollarBrace    bool
	ConfigHasDoubleDollar         bool
	ConfigHasBackslashDollarBrace bool
	ConfigHasBackslashDollar      bool

	StatePathIsShared        bool
	StatePathInDeployerHome  bool
	StatePathInOtherUserHome bool
	StatePathOther           bool

	// Independent booleans, but a false is not lost as a single bit: they are
	// always co-emitted with the one-hot DMSCompat verdict below, whose presence
	// signals the measurement ran, so a 0 here reads as "measured false".
	DMSUndeclaredDeployingUser    bool
	DMSUndeclaredOtherUser        bool
	DMSUndeclaredServicePrincipal bool
	DMSUndeclaredGroup            bool

	DMSCompatAuto               bool
	DMSCompatOnlySelfUndeclared bool
	DMSCompatNot                bool

	DirectMigrateError           bool
	DirectMigrateWarnings        bool
	DirectMigratePlanError       bool
	DirectMigrateRecreatePlanned bool
	DirectMigratedViaConfig      bool
	DirectMigratedViaEnv         bool
	DirectMigratedViaDefault     bool

	EngineTerraformConfig bool
	EngineTerraformEnv    bool

	// Measured flags: see BoolPair.
	SkipArtifactCleanup              BoolPair
	HasServerlessCompute             BoolPair
	HasClassicJobCompute             BoolPair
	HasClassicInteractiveCompute     BoolPair
	ArtifactBuildCommandIsSet        BoolPair
	ArtifactFilesIsSet               BoolPair
	SourceLinkedSetForNonDevelopment BoolPair
	PresetsNamePrefixIsSet           BoolPair
	UseLegacyRunAs                   BoolPair
	RunAsSet                         BoolPair
	PermissionsSectionSet            BoolPair
	PythonWheelWrapperIsSet          BoolPair
	AppLifecycleStarted              BoolPair
	ClusterLifecycleStarted          BoolPair
	SqlWarehouseLifecycleStarted     BoolPair
	DeploymentHistoryEnabled         BoolPair
	ImmutableSnapshotUsed            BoolPair
	AiRuntimeTaskScheduled           BoolPair
	AiRuntimeTaskMultitask           BoolPair
}

// BoolPair records a measured boolean as two presence bits, so the bitmap can
// tell a measured false from a flag that was never measured (neither set). Both
// are set when the flag was measured more than once with different outcomes,
// e.g. once per artifact.
type BoolPair struct {
	True  bool
	False bool
}

// Set records one measurement.
func (p *BoolPair) Set(value bool) {
	if value {
		p.True = true
	} else {
		p.False = true
	}
}
