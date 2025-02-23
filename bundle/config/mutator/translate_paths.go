package mutator

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/notebook"
)

// TranslateMode specifies how a path should be translated.
type TranslateMode int

const (
	// TranslateModeNotebook translates a path to a remote notebook.
	TranslateModeNotebook TranslateMode = iota

	// TranslateModeFile translates a path to a remote regular file.
	TranslateModeFile

	// TranslateModeDirectory translates a path to a remote directory.
	TranslateModeDirectory

	// TranslateModeLocalAbsoluteFile translates a path to the local absolute file path.
	// It returns an error if the path does not exist or is a directory.
	TranslateModeLocalAbsoluteFile

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
)

// translateOptions control path translation behavior.
type translateOptions struct {
	// Mode specifies how the path should be translated.
	Mode TranslateMode

	// AllowPathOutsideSyncRoot can be set for paths that are not tied to the sync root path.
	// This is the case for artifact paths, for example.
	AllowPathOutsideSyncRoot bool
}

type ErrIsNotebook struct {
	path string
}

func (err ErrIsNotebook) Error() string {
	return fmt.Sprintf("file at %s is a notebook", err.path)
}

type ErrIsNotNotebook struct {
	path string
}

func (err ErrIsNotNotebook) Error() string {
	return fmt.Sprintf("file at %s is not a notebook", err.path)
}

type translatePaths struct{}

// TranslatePaths converts paths to local notebook files into paths in the workspace file system.
func TranslatePaths() bundle.Mutator {
	return &translatePaths{}
}

func (m *translatePaths) Name() string {
	return "TranslatePaths"
}

// translateContext is a context for rewriting paths in a config.
// It is freshly instantiated on every mutator apply call.
// It provides access to the underlying bundle object such that
// it doesn't have to be passed around explicitly.
type translateContext struct {
	b *bundle.Bundle

	// seen is a map of local paths to their corresponding remote paths.
	// If a local path has already been successfully resolved, we do not need to resolve it again.
	seen map[string]string

	// remoteRoot is the root path of the remote workspace.
	// It is equal to ${workspace.file_path} for regular deployments.
	// It points to the source root path for source-linked deployments.
	remoteRoot string
}

// rewritePath converts a given relative path from the loaded config to a new path based on the passed rewriting function
//
// It takes these arguments:
//   - The context in which the function is called.
//   - The argument `dir` is the directory relative to which the relative path should be interpreted.
//   - The argument `input` is the relative path to rewrite.
//   - The argument `opts` is a struct that specifies how the path should be rewritten.
//     It contains a `Mode` field that specifies how the path should be rewritten.
//
// The function returns the rewritten path if successful, or an error if the path could not be rewritten.
// The returned path is an empty string if the path was not rewritten.
func (t *translateContext) rewritePath(
	ctx context.Context,
	dir string,
	input string,
	opts translateOptions,
) (string, error) {
	// We assume absolute paths point to a location in the workspace
	if path.IsAbs(input) {
		return "", nil
	}

	url, err := url.Parse(input)
	if err != nil {
		return "", err
	}

	// If the file path has scheme, it's a full path and we don't need to transform it
	if url.Scheme != "" {
		return "", nil
	}

	// Local path is relative to the directory the resource was defined in.
	localPath := filepath.Join(dir, input)
	if interp, ok := t.seen[localPath]; ok {
		return interp, nil
	}

	// Local path must be contained in the sync root.
	// If it isn't, it won't be synchronized into the workspace.
	localRelPath, err := filepath.Rel(t.b.SyncRootPath, localPath)
	if err != nil {
		return "", err
	}
	if !opts.AllowPathOutsideSyncRoot && !filepath.IsLocal(localRelPath) {
		return "", fmt.Errorf("path %s is not contained in sync root path", localPath)
	}

	// Normalize paths to separated by forward slashes.
	localPath = filepath.ToSlash(localPath)
	localRelPath = filepath.ToSlash(localRelPath)

	// Convert local path into workspace path via specified function.
	var interp string
	switch opts.Mode {
	case TranslateModeNotebook:
		interp, err = t.translateNotebookPath(ctx, input, localPath, localRelPath)
	case TranslateModeFile:
		interp, err = t.translateFilePath(ctx, input, localPath, localRelPath)
	case TranslateModeDirectory:
		interp, err = t.translateDirectoryPath(ctx, input, localPath, localRelPath)
	case TranslateModeLocalAbsoluteFile:
		interp, err = t.translateLocalAbsoluteFilePath(ctx, input, localPath, localRelPath)
	case TranslateModeLocalAbsoluteDirectory:
		interp, err = t.translateLocalAbsoluteDirectoryPath(ctx, input, localPath, localRelPath)
	case TranslateModeLocalRelative:
		interp, err = t.translateLocalRelativePath(ctx, input, localPath, localRelPath)
	case TranslateModeLocalRelativeWithPrefix:
		interp, err = t.translateLocalRelativeWithPrefixPath(ctx, input, localPath, localRelPath)
	default:
		return "", fmt.Errorf("unsupported translate mode: %d", opts.Mode)
	}
	if err != nil {
		return "", err
	}

	t.seen[localPath] = interp
	return interp, nil
}

func (t *translateContext) translateNotebookPath(ctx context.Context, literal, localFullPath, localRelPath string) (string, error) {
	nb, _, err := notebook.DetectWithFS(t.b.SyncRoot, localRelPath)
	if errors.Is(err, fs.ErrNotExist) {
		if path.Ext(localFullPath) != notebook.ExtensionNone {
			return "", fmt.Errorf("notebook %s not found", literal)
		}

		extensions := []string{
			notebook.ExtensionPython,
			notebook.ExtensionR,
			notebook.ExtensionScala,
			notebook.ExtensionSql,
			notebook.ExtensionJupyter,
		}

		// Check whether a file with a notebook extension already exists. This
		// way we can provide a more targeted error message.
		for _, ext := range extensions {
			literalWithExt := literal + ext
			localRelPathWithExt := localRelPath + ext
			if _, err := fs.Stat(t.b.SyncRoot, localRelPathWithExt); err == nil {
				return "", fmt.Errorf(`notebook %s not found. Did you mean %s?
Local notebook references are expected to contain one of the following
file extensions: [%s]`, literal, literalWithExt, strings.Join(extensions, ", "))
			}
		}

		// Return a generic error message if no matching possible file is found.
		return "", fmt.Errorf(`notebook %s not found. Local notebook references are expected
to contain one of the following file extensions: [%s]`, literal, strings.Join(extensions, ", "))
	}
	if err != nil {
		return "", fmt.Errorf("unable to determine if %s is a notebook: %w", localFullPath, err)
	}
	if !nb {
		return "", ErrIsNotNotebook{localFullPath}
	}

	// Upon import, notebooks are stripped of their extension.
	localRelPathNoExt := strings.TrimSuffix(localRelPath, path.Ext(localRelPath))
	return path.Join(t.remoteRoot, localRelPathNoExt), nil
}

func (t *translateContext) translateFilePath(ctx context.Context, literal, localFullPath, localRelPath string) (string, error) {
	nb, _, err := notebook.DetectWithFS(t.b.SyncRoot, localRelPath)
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("file %s not found", literal)
	}
	if err != nil {
		return "", fmt.Errorf("unable to determine if %s is not a notebook: %w", filepath.FromSlash(localFullPath), err)
	}
	if nb {
		return "", ErrIsNotebook{localFullPath}
	}
	return path.Join(t.remoteRoot, localRelPath), nil
}

func (t *translateContext) translateDirectoryPath(ctx context.Context, literal, localFullPath, localRelPath string) (string, error) {
	info, err := t.b.SyncRoot.Stat(localRelPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", filepath.FromSlash(localFullPath))
	}
	return path.Join(t.remoteRoot, localRelPath), nil
}

func (t *translateContext) translateLocalAbsoluteFilePath(ctx context.Context, literal, localFullPath, localRelPath string) (string, error) {
	info, err := t.b.SyncRoot.Stat(localRelPath)
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("file %s not found", literal)
	}
	if err != nil {
		return "", fmt.Errorf("unable to determine if %s is a file: %w", filepath.FromSlash(localFullPath), err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("expected %s to be a file but found a directory", literal)
	}
	return localFullPath, nil
}

func (t *translateContext) translateLocalAbsoluteDirectoryPath(ctx context.Context, literal, localFullPath, _ string) (string, error) {
	info, err := os.Stat(filepath.FromSlash(localFullPath))
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("directory %s not found", literal)
	}
	if err != nil {
		return "", fmt.Errorf("unable to determine if %s is a directory: %w", filepath.FromSlash(localFullPath), err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("expected %s to be a directory but found a file", literal)
	}
	return localFullPath, nil
}

func (t *translateContext) translateLocalRelativePath(ctx context.Context, literal, localFullPath, localRelPath string) (string, error) {
	return localRelPath, nil
}

func (t *translateContext) translateLocalRelativeWithPrefixPath(ctx context.Context, literal, localFullPath, localRelPath string) (string, error) {
	if !strings.HasPrefix(localRelPath, ".") {
		localRelPath = "./" + localRelPath
	}
	return localRelPath, nil
}

func (t *translateContext) rewriteValue(ctx context.Context, p dyn.Path, v dyn.Value, dir string, opts translateOptions) (dyn.Value, error) {
	out, err := t.rewritePath(ctx, dir, v.MustString(), opts)
	if err != nil {
		if target := (&ErrIsNotebook{}); errors.As(err, target) {
			return dyn.InvalidValue, fmt.Errorf(`expected a file for "%s" but got a notebook: %w`, p, target)
		}
		if target := (&ErrIsNotNotebook{}); errors.As(err, target) {
			return dyn.InvalidValue, fmt.Errorf(`expected a notebook for "%s" but got a file: %w`, p, target)
		}
		return dyn.InvalidValue, err
	}

	// If the path was not rewritten, return the original value.
	if out == "" {
		return v, nil
	}

	return dyn.NewValue(out, v.Locations()), nil
}

func (m *translatePaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	t := &translateContext{
		b:    b,
		seen: make(map[string]string),
	}

	// Set the remote root to the sync root if source-linked deployment is enabled.
	// Otherwise, set it to the workspace file path.
	if config.IsExplicitlyEnabled(t.b.Config.Presets.SourceLinkedDeployment) {
		t.remoteRoot = t.b.SyncRootPath
	} else {
		t.remoteRoot = t.b.Config.Workspace.FilePath
	}

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		var err error
		for _, fn := range []func(context.Context, dyn.Value) (dyn.Value, error){
			t.applyJobTranslations,
			t.applyPipelineTranslations,
			t.applyArtifactTranslations,
			t.applyDashboardTranslations,
			t.applyAppsTranslations,
		} {
			v, err = fn(ctx, v)
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
		return v, nil
	})

	return diag.FromErr(err)
}

// gatherFallbackPaths collects the fallback paths for relative paths in the configuration.
// Read more about the motivation for this functionality in the "fallback" path translation tests.
func gatherFallbackPaths(v dyn.Value, typ string) (map[string]string, error) {
	fallback := make(map[string]string)
	pattern := dyn.NewPattern(dyn.Key("resources"), dyn.Key(typ), dyn.AnyKey())

	// Previous behavior was to use a resource's location as the base path to resolve
	// relative paths in its definition. With the introduction of [dyn.Value] throughout,
	// we can use the location of the [dyn.Value] of the relative path itself.
	//
	// This is more flexible, as resources may have overrides that are not
	// located in the same directory as the resource configuration file.
	//
	// To maintain backwards compatibility, we allow relative paths to be resolved using
	// the original approach as fallback if the [dyn.Value] location cannot be resolved.
	_, err := dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		key := p[2].Key()
		dir, err := v.Location().Directory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for %s: %w", p, err)
		}
		fallback[key] = dir
		return v, nil
	})
	if err != nil {
		return nil, err
	}
	return fallback, nil
}
