package mutator

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/notebook"
)

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

type rewriteFunc func(literal, localFullPath, localRelPath, remotePath string) (string, error)

// translateContext is a context for rewriting paths in a config.
// It is freshly instantiated on every mutator apply call.
// It provides access to the underlying bundle object such that
// it doesn't have to be passed around explicitly.
type translateContext struct {
	b *bundle.Bundle

	// seen is a map of local paths to their corresponding remote paths.
	// If a local path has already been successfully resolved, we do not need to resolve it again.
	seen map[string]string
}

// GetLocalPath returns the local file system paths for a path referenced from a resource..
// If it's an absolute path, we treat it as a workspace path and return "".
//
// Arguments:
//
//	sourceDir - the source directory for the resource definition.
//	p         - the path referenced from the resource definition.
//
// Returns:
//
//	localPath - the full local file system path.
//	localRelPath  - the relative path from the base directory.
func GetLocalPath(ctx context.Context, b *bundle.Bundle, sourceDir, p string) (string, string, error) {
	if p == "" {
		return "", "", fmt.Errorf("path cannot be empty")
	}
	if path.IsAbs(p) {
		return "", "", nil
	}

	url, err := url.Parse(p)
	if err != nil {
		// Apparently this path is not a URL; this can happen for paths that
		// have non-URL characters like './profit-%.csv'.
		log.Warnf(ctx, "Failed to parse path as a URL '%s': %v", p, err)
	} else if url.Scheme != "" {
		return "", "", nil
	}

	localPath := filepath.Join(sourceDir, filepath.FromSlash(p))
	localRelPath, err := filepath.Rel(b.SyncRootPath, localPath)
	if err != nil {
		return "", "", err
	}
	if strings.HasPrefix(localRelPath, "..") {
		return "", "", fmt.Errorf("path '%s' is not contained in sync root path", p)
	}

	return localPath, localRelPath, nil
}

// rewritePath converts a given relative path from the loaded config to a new path based on the passed rewriting function
//
// It takes these arguments:
//   - The argument `dir` is the directory relative to which the given relative path is.
//   - The given relative path is both passed and written back through `*p`.
//   - The argument `fn` is a function that performs the actual rewriting logic.
//     This logic is different between regular files or notebooks.
//
// The function returns an error if it is impossible to rewrite the given relative path.
func (t *translateContext) rewritePath(
	ctx context.Context,
	dir string,
	p *string,
	fn rewriteFunc,
) error {
	localPath, localRelPath, err := GetLocalPath(ctx, t.b, dir, *p)
	if err != nil {
		return err
	}
	if localPath == "" {
		// Skip absolute paths
		return nil
	}

	if interp, ok := t.seen[localPath]; ok {
		*p = interp
		return nil
	}

	var workspacePath string
	if config.IsExplicitlyEnabled(t.b.Config.Presets.SourceLinkedDeployment) {
		workspacePath = t.b.SyncRootPath
	} else {
		workspacePath = t.b.Config.Workspace.FilePath
	}
	remotePath := path.Join(workspacePath, filepath.ToSlash(localRelPath))

	// Convert local path into workspace path via specified function.
	interp, err := fn(*p, localPath, localRelPath, remotePath)
	if err != nil {
		return err
	}

	*p = interp
	t.seen[localPath] = interp
	return nil
}

func (t *translateContext) translateNotebookPath(literal, localFullPath, localRelPath, remotePath string) (string, error) {
	nb, _, err := notebook.DetectWithFS(t.b.SyncRoot, filepath.ToSlash(localRelPath))
	if errors.Is(err, fs.ErrNotExist) {
		if filepath.Ext(localFullPath) != notebook.ExtensionNone {
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
			localRelPathWithExt := filepath.ToSlash(localRelPath + ext)
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
	return strings.TrimSuffix(remotePath, filepath.Ext(localFullPath)), nil
}

func (t *translateContext) translateFilePath(literal, localFullPath, localRelPath, remotePath string) (string, error) {
	nb, _, err := notebook.DetectWithFS(t.b.SyncRoot, filepath.ToSlash(localRelPath))
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("file %s not found", literal)
	}
	if err != nil {
		return "", fmt.Errorf("unable to determine if %s is not a notebook: %w", localFullPath, err)
	}
	if nb {
		return "", ErrIsNotebook{localFullPath}
	}
	return remotePath, nil
}

func (t *translateContext) translateDirectoryPath(literal, localFullPath, localRelPath, remotePath string) (string, error) {
	info, err := t.b.SyncRoot.Stat(filepath.ToSlash(localRelPath))
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", localFullPath)
	}
	return remotePath, nil
}

func (t *translateContext) translateNoOp(literal, localFullPath, localRelPath, remotePath string) (string, error) {
	return localRelPath, nil
}

func (t *translateContext) retainLocalAbsoluteFilePath(literal, localFullPath, localRelPath, remotePath string) (string, error) {
	info, err := t.b.SyncRoot.Stat(filepath.ToSlash(localRelPath))
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("file %s not found", literal)
	}
	if err != nil {
		return "", fmt.Errorf("unable to determine if %s is a file: %w", localFullPath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("expected %s to be a file but found a directory", literal)
	}
	return localFullPath, nil
}

func (t *translateContext) translateNoOpWithPrefix(literal, localFullPath, localRelPath, remotePath string) (string, error) {
	if !strings.HasPrefix(localRelPath, ".") {
		localRelPath = "." + string(filepath.Separator) + localRelPath
	}
	return localRelPath, nil
}

func (t *translateContext) rewriteValue(ctx context.Context, p dyn.Path, v dyn.Value, fn rewriteFunc, dir string) (dyn.Value, error) {
	out := v.MustString()
	err := t.rewritePath(ctx, dir, &out, fn)
	if err != nil {
		if target := (&ErrIsNotebook{}); errors.As(err, target) {
			return dyn.InvalidValue, fmt.Errorf(`expected a file for "%s" but got a notebook: %w`, p, target)
		}
		if target := (&ErrIsNotNotebook{}); errors.As(err, target) {
			return dyn.InvalidValue, fmt.Errorf(`expected a notebook for "%s" but got a file: %w`, p, target)
		}
		return dyn.InvalidValue, err
	}

	return dyn.NewValue(out, v.Locations()), nil
}

func (t *translateContext) rewriteRelativeTo(ctx context.Context, p dyn.Path, v dyn.Value, fn rewriteFunc, dir, fallback string) (dyn.Value, error) {
	nv, err := t.rewriteValue(ctx, p, v, fn, dir)
	if err == nil {
		return nv, nil
	}

	// If we failed to rewrite the path, try to rewrite it relative to the fallback directory.
	if fallback != "" {
		nv, nerr := t.rewriteValue(ctx, p, v, fn, fallback)
		if nerr == nil {
			// TODO: Emit a warning that this path should be rewritten.
			return nv, nil
		}
	}

	return dyn.InvalidValue, err
}

func (m *translatePaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	t := &translateContext{
		b:    b,
		seen: make(map[string]string),
	}

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		var err error
		for _, fn := range []func(ctx context.Context, v dyn.Value) (dyn.Value, error){
			t.applyJobTranslations,
			t.applyPipelineTranslations,
			t.applyArtifactTranslations,
			t.applyDashboardTranslations,
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
