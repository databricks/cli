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
// measured") are stored as a True/False pair so the presence-only bitmap can
// represent both outcomes: neither set means "not measured".
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

	DMSUndeclaredDeployingUser    bool
	DMSUndeclaredOtherUser        bool
	DMSUndeclaredServicePrincipal bool
	DMSUndeclaredGroup            bool

	DMSCompatAuto               bool
	DMSCompatOnlySelfUndeclared bool
	DMSCompatNot                bool

	DirectMigrateError       bool
	DirectMigrateCommitError bool
	DirectMigrateWarnings    bool
	DirectMigratedViaConfig  bool
	DirectMigratedViaEnv     bool

	EngineTerraformConfig bool
	EngineTerraformEnv    bool

	// True/False pairs: exactly one is set when the flag is measured, neither
	// when it is not.
	SkipArtifactCleanupTrue  bool
	SkipArtifactCleanupFalse bool

	HasServerlessComputeTrue  bool
	HasServerlessComputeFalse bool

	HasClassicJobComputeTrue  bool
	HasClassicJobComputeFalse bool

	HasClassicInteractiveComputeTrue  bool
	HasClassicInteractiveComputeFalse bool

	ArtifactBuildCommandIsSetTrue  bool
	ArtifactBuildCommandIsSetFalse bool

	ArtifactFilesIsSetTrue  bool
	ArtifactFilesIsSetFalse bool

	SourceLinkedSetForNonDevelopmentTrue  bool
	SourceLinkedSetForNonDevelopmentFalse bool

	PresetsNamePrefixIsSetTrue  bool
	PresetsNamePrefixIsSetFalse bool

	UseLegacyRunAsTrue  bool
	UseLegacyRunAsFalse bool

	RunAsSetTrue  bool
	RunAsSetFalse bool

	PermissionsSectionSetTrue  bool
	PermissionsSectionSetFalse bool

	PythonWheelWrapperIsSetTrue  bool
	PythonWheelWrapperIsSetFalse bool

	AppLifecycleStartedTrue  bool
	AppLifecycleStartedFalse bool

	ClusterLifecycleStartedTrue  bool
	ClusterLifecycleStartedFalse bool

	SqlWarehouseLifecycleStartedTrue  bool
	SqlWarehouseLifecycleStartedFalse bool

	DirectDryMigrateSuccessTrue  bool
	DirectDryMigrateSuccessFalse bool

	DirectDryMigrateWarningsTrue  bool
	DirectDryMigrateWarningsFalse bool
}

// SetPaired sets exactly one bool of a True/False pair from a measured value,
// e.g. b.Telemetry.SetPaired(&t.RunAsSetTrue, &t.RunAsSetFalse, value).
func (t *Telemetry) SetPaired(vTrue, vFalse *bool, value bool) {
	if value {
		*vTrue = true
	} else {
		*vFalse = true
	}
}
