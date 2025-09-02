package paths

// TranslateMode specifies how a path should be translated.
type TranslateMode int

const (
	// TranslateModeNotebook translates a path to a remote notebook.
	TranslateModeNotebook TranslateMode = iota

	// TranslateModeFile translates a path to a remote regular file.
	TranslateModeFile

	// TranslateModeDirectory translates a path to a remote directory.
	TranslateModeDirectory

	// TranslateModeGlob translates a relative glob pattern to a remote glob pattern.
	// It does not perform any checks on the glob pattern itself.
	TranslateModeGlob

	// TranslateModeLocalAbsoluteDirectory translates a path to the local absolute directory path.
	// It returns an error if the path does not exist or is not a directory.
	TranslateModeLocalAbsoluteDirectory

	// TranslateModeLocalRelative translates a path to be relative to the bundle sync root path.
	// It does not check if the path exists, nor care if it is a file or directory.
	TranslateModeLocalRelative

	// TranslateModeLocalRelativeWithPrefix translates a path to be relative to the bundle sync root path.
	// It a "./" prefix to the path if it does not already have one.
	// This allows for disambiguating between paths and PyPI package names.
	TranslateModeLocalRelativeWithPrefix

	// TranslateModeEnvironmentRequirements translates a local requirements file path to be absolute.
	TranslateModeEnvironmentRequirements
)
