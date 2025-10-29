package template

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/cli/libs/log"
)

type Reader interface {
	// LoadSchemaAndTemplateFS loads and returns the schema and template filesystem.
	LoadSchemaAndTemplateFS(ctx context.Context) (*jsonschema.Schema, fs.FS, error)

	// Cleanup releases any resources associated with the reader
	// like cleaning up temporary directories.
	Cleanup(ctx context.Context)
}

// builtinReader reads a template from the built-in templates.
type builtinReader struct {
	name string
}

func (r *builtinReader) LoadSchemaAndTemplateFS(ctx context.Context) (*jsonschema.Schema, fs.FS, error) {
	builtin, err := builtin()
	if err != nil {
		return nil, nil, err
	}

	var schemaFS fs.FS
	for _, entry := range builtin {
		if entry.Name == r.name {
			schemaFS = entry.FS
			break
		}
	}

	if schemaFS == nil {
		return nil, nil, fmt.Errorf("builtin template %s not found", r.name)
	}

	schema, err := jsonschema.LoadFS(schemaFS, schemaFileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("not a bundle template: expected to find a template schema file at %s", schemaFileName)
		}
		return nil, nil, fmt.Errorf("failed to load schema for template %s: %w", r.name, err)
	}

	// If no template_dir is specified, assume it's in the same directory as the schema
	if schema.TemplateDir == "" {
		return schema, schemaFS, nil
	}

	// Find the referenced template filesystem
	templateDirName := filepath.Base(schema.TemplateDir)
	for _, entry := range builtin {
		if entry.Name == templateDirName {
			return schema, entry.FS, nil
		}
	}

	return nil, nil, fmt.Errorf("template directory %s (referenced by %s) not found", templateDirName, r.name)
}

func (r *builtinReader) Cleanup(ctx context.Context) {}

// gitReader reads a template from a git repository.
type gitReader struct {
	gitUrl string
	// tag or branch to checkout
	ref string
	// subdirectory within the repository that contains the template
	templateDir string
	// temporary directory where the repository is cloned
	tmpRepoDir string

	// Function to clone the repository. This is a function pointer to allow
	// mocking in tests.
	cloneFunc func(ctx context.Context, url, reference, targetPath string) error
}

// Computes the repo name from the repo URL. Treats the last non empty word
// when splitting at '/' as the repo name. For example: for url git@github.com:databricks/cli.git
// the name would be "cli.git"
func repoName(url string) string {
	parts := strings.Split(strings.TrimRight(url, "/"), "/")
	return parts[len(parts)-1]
}

func (r *gitReader) LoadSchemaAndTemplateFS(ctx context.Context) (*jsonschema.Schema, fs.FS, error) {
	// Calling LoadSchemaAndTemplateFS twice will lead to two downloaded copies of the git repo.
	// In the future if you need to call this twice, consider adding some caching
	// logic here to avoid multiple downloads.
	if r.tmpRepoDir != "" {
		return nil, nil, errors.New("LoadSchemaAndTemplateFS called twice on git reader")
	}

	// Create a temporary directory with the name of the repository.  The '*'
	// character is replaced by a random string in the generated temporary directory.
	repoDir, err := os.MkdirTemp("", repoName(r.gitUrl)+"-*")
	if err != nil {
		return nil, nil, err
	}
	r.tmpRepoDir = repoDir

	// start the spinner
	promptSpinner := cmdio.Spinner(ctx)
	promptSpinner <- "Downloading the template\n"

	err = r.cloneFunc(ctx, r.gitUrl, r.ref, repoDir)
	close(promptSpinner)
	if err != nil {
		return nil, nil, err
	}

	templateDir := filepath.Join(repoDir, r.templateDir)
	return loadSchemaAndResolveTemplateDir(templateDir)
}

func (r *gitReader) Cleanup(ctx context.Context) {
	if r.tmpRepoDir == "" {
		return
	}

	// Cleanup is best effort. Only log errors.
	err := os.RemoveAll(r.tmpRepoDir)
	if err != nil {
		log.Debugf(ctx, "Error cleaning up tmp directory %s for git template reader for URL %s: %s", r.tmpRepoDir, r.gitUrl, err)
	}
}

// localReader reads a template from a local filesystem.
type localReader struct {
	// Path on the local filesystem that contains the template
	path string
}

func (r *localReader) LoadSchemaAndTemplateFS(ctx context.Context) (*jsonschema.Schema, fs.FS, error) {
	return loadSchemaAndResolveTemplateDir(r.path)
}

func (r *localReader) Cleanup(ctx context.Context) {}

// loadSchemaAndResolveTemplateDir loads a schema from a local directory path
// and resolves any template_dir reference.
func loadSchemaAndResolveTemplateDir(path string) (*jsonschema.Schema, fs.FS, error) {
	templateFS := os.DirFS(path)
	schema, err := jsonschema.LoadFS(templateFS, schemaFileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("not a bundle template: expected to find a template schema file at %s", schemaFileName)
		}
		return nil, nil, fmt.Errorf("failed to load schema: %w", err)
	}

	// If no template_dir is specified, just use templateFS
	if schema.TemplateDir == "" {
		return schema, templateFS, nil
	}

	// Resolve template_dir relative to the schema location
	templateDir := filepath.Join(path, schema.TemplateDir)

	// Check if the referenced template directory exists
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("template directory %s not found", templateDir)
	}

	return schema, os.DirFS(templateDir), nil
}
