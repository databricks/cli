package mutator

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
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

type translatePaths struct {
	seen map[string]string
}

// TranslatePaths converts paths to local notebook files into paths in the workspace file system.
func TranslatePaths() bundle.Mutator {
	return &translatePaths{}
}

func (m *translatePaths) Name() string {
	return "TranslatePaths"
}

type rewriteFunc func(literal, localFullPath, localRelPath, remotePath string) (string, error)

// rewritePath converts a given relative path from the loaded config to a new path based on the passed rewriting function
//
// It takes these arguments:
//   - The argument `dir` is the directory relative to which the given relative path is.
//   - The given relative path is both passed and written back through `*p`.
//   - The argument `fn` is a function that performs the actual rewriting logic.
//     This logic is different between regular files or notebooks.
//
// The function returns an error if it is impossible to rewrite the given relative path.
func (m *translatePaths) rewritePath(
	dir string,
	b *bundle.Bundle,
	p *string,
	fn rewriteFunc,
) error {
	// We assume absolute paths point to a location in the workspace
	if path.IsAbs(filepath.ToSlash(*p)) {
		return nil
	}

	url, err := url.Parse(*p)
	if err != nil {
		return err
	}

	// If the file path has scheme, it's a full path and we don't need to transform it
	if url.Scheme != "" {
		return nil
	}

	// Local path is relative to the directory the resource was defined in.
	localPath := filepath.Join(dir, filepath.FromSlash(*p))
	if interp, ok := m.seen[localPath]; ok {
		*p = interp
		return nil
	}

	// Remote path must be relative to the bundle root.
	localRelPath, err := filepath.Rel(b.Config.Path, localPath)
	if err != nil {
		return err
	}
	if strings.HasPrefix(localRelPath, "..") {
		return fmt.Errorf("path %s is not contained in bundle root path", localPath)
	}

	// Prefix remote path with its remote root path.
	remotePath := path.Join(b.Config.Workspace.FilesPath, filepath.ToSlash(localRelPath))

	// Convert local path into workspace path via specified function.
	interp, err := fn(*p, localPath, localRelPath, filepath.ToSlash(remotePath))
	if err != nil {
		return err
	}

	*p = interp
	m.seen[localPath] = interp
	return nil
}

func translateNotebookPath(literal, localFullPath, localRelPath, remotePath string) (string, error) {
	nb, _, err := notebook.Detect(localFullPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("notebook %s not found", literal)
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

func translateFilePath(literal, localFullPath, localRelPath, remotePath string) (string, error) {
	nb, _, err := notebook.Detect(localFullPath)
	if os.IsNotExist(err) {
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

func translateNoOp(literal, localFullPath, localRelPath, remotePath string) (string, error) {
	return localRelPath, nil
}

type transformer struct {
	// A directory path relative to which `path` will be transformed
	dir string
	// A path to transform
	path *string
	// Name of the config property where the path string is coming from
	configPath string
	// A function that performs the actual rewriting logic.
	fn rewriteFunc
}

type transformFunc func(resource any, dir string) *transformer

// Apply all matches transformers for the given resource
func (m *translatePaths) applyTransformers(funcs []transformFunc, b *bundle.Bundle, resource any, dir string) error {
	for _, transformFn := range funcs {
		transformer := transformFn(resource, dir)
		if transformer == nil {
			continue
		}

		err := m.rewritePath(transformer.dir, b, transformer.path, transformer.fn)
		if err != nil {
			if target := (&ErrIsNotebook{}); errors.As(err, target) {
				return fmt.Errorf(`expected a file for "%s" but got a notebook: %w`, transformer.configPath, target)
			}
			if target := (&ErrIsNotNotebook{}); errors.As(err, target) {
				return fmt.Errorf(`expected a notebook for "%s" but got a file: %w`, transformer.configPath, target)
			}
			return err
		}
	}

	return nil
}

func (m *translatePaths) Apply(_ context.Context, b *bundle.Bundle) error {
	m.seen = make(map[string]string)

	for _, fn := range []func(*translatePaths, *bundle.Bundle) error{
		applyJobTransformers,
		applyPipelineTransformers,
		applyArtifactTransformers,
	} {
		err := fn(m, b)
		if err != nil {
			return err
		}
	}

	return nil
}
