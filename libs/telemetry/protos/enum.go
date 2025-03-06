package protos

type DummyCliEnum string

const (
	DummyCliEnumUnspecified DummyCliEnum = "DUMMY_CLI_ENUM_UNSPECIFIED"
	DummyCliEnumValue1      DummyCliEnum = "VALUE1"
	DummyCliEnumValue2      DummyCliEnum = "VALUE2"
	DummyCliEnumValue3      DummyCliEnum = "VALUE3"
)

type BundleMode string

const (
	BundleModeUnspecified BundleMode = "TYPE_UNSPECIFIED"
	BundleModeDevelopment BundleMode = "DEVELOPMENT"
	BundleModeProduction  BundleMode = "PRODUCTION"
)

type BundleDeployArtifactPathType string

const (
	BundleDeployArtifactPathTypeUnspecified BundleDeployArtifactPathType = "TYPE_UNSPECIFIED"
	BundleDeployArtifactPathTypeWorkspace   BundleDeployArtifactPathType = "WORKSPACE_FILE_SYSTEM"
	BundleDeployArtifactPathTypeVolume      BundleDeployArtifactPathType = "UC_VOLUME"
)
