package template

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/exp/slices"
)

// This structure renders any template files during project initialization
type renderer struct {
	// A config that is the "dot" value available to any template being rendered.
	// Refer to https://pkg.go.dev/text/template for how templates can use
	// this "dot" value
	config map[string]any

	// A base template with helper functions and user defined template in the
	// library directory loaded. This is used as the base to compute any project
	// templates during file tree walk
	baseTemplate *template.Template
}

func newRenderer(config map[string]any, libraryRoot string) (*renderer, error) {
	// All user defined functions will be available inside library root
	libraryGlob := filepath.Join(libraryRoot, "*")

	// Initialize new template, with helper functions loaded
	tmpl := template.New("").Funcs(HelperFuncs)

	// Load files in the library to the template
	matches, err := filepath.Glob(libraryGlob)
	if err != nil {
		return nil, err
	}
	if len(matches) != 0 {
		tmpl, err = tmpl.ParseGlob(libraryGlob)
		if err != nil {
			return nil, err
		}
	}

	return &renderer{
		config:       config,
		baseTemplate: tmpl,
	}, nil
}

// Executes the template by applying config on it. Returns the materialized template
// as a string
func (r *renderer) executeTemplate(templateDefinition string) (string, error) {
	// Create copy of base template so as to not overwrite it
	tmpl, err := r.baseTemplate.Clone()
	if err != nil {
		return "", err
	}

	// Parse the template text
	tmpl, err = tmpl.Parse(templateDefinition)
	if err != nil {
		return "", err
	}

	// Execute template and get result
	result := strings.Builder{}
	err = tmpl.Execute(&result, r.config)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func (r *renderer) generateFile(pathTemplate, contentTemplate string, perm fs.FileMode) (*inMemoryFile, error) {
	// compute file content
	fileContent, err := r.executeTemplate(contentTemplate)
	if errors.Is(err, errSkipThisFile) {
		// skip this file
		return nil, nil
	}

	// Capture errors caused by the "fail" helper function
	if target := (&ErrFail{}); errors.As(err, target) {
		return nil, target
	}
	if err != nil {
		return nil, fmt.Errorf("failed to compute file content for %s. %w", pathTemplate, err)
	}

	// compute the path for this file
	path, err := r.executeTemplate(pathTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to compute path for %s. %w", pathTemplate, err)
	}

	return &inMemoryFile{
		path:    path,
		content: fileContent,
		perm:    perm,
	}, nil
}

type inMemoryFile struct {
	path    string
	// TODO: use bytes in to serialize for binary files, Can we just use string here, is it the same
	content string
	perm    fs.FileMode
}

func (r *renderer) generateDir(templateDir, instanceDir string) (map[*inMemoryFile]any, error) {
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return nil, err
	}

	// Args from any calls to the {{skip}} helper function will be appended to this list.
	skipPatterns := make([]string, 0)
	// Add skip functional closure which uses the newly initialized slice to store patterns
	r.baseTemplate.Funcs(template.FuncMap{
		"skip": func(pattern string) error {
			if !slices.Contains(skipPatterns, pattern) {
				skipPatterns = append(skipPatterns, pattern)
			}
			return nil
		},
	})

	// Initialize set to contain in memory files
	files := make(map[*inMemoryFile]any, 0)

	// TODO: use slice and pure functions for skip computation
	// TODO: Have a global list of skip patterns accumulated for the entire file tree
	// TODO: Write file tree to disk all at once
	// TODO: optimization skip subdirectoies if we already already can

	// Compute the in memory file representation for the executed template
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// read template file to get the templatized content for the file
		b, err := os.ReadFile(filepath.Join(templateDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		contentTemplate := string(b)

		// Generate an in memory representation of the file, by executing the template
		f, err := r.generateFile(filepath.Join(instanceDir, entry.Name()), contentTemplate, entry.Type().Perm())
		if err != nil {
			return nil, err
		}
		files[f] = nil
	}

	// Match glob patterns stored, and delete matching in memory files
	return nil, deleteSkippedFiles(files, skipPatterns)
}

// deletes any files in [files] whose name matches a glob pattern in [skipPatterns]
func deleteSkippedFiles(files map[*inMemoryFile]any, skipPatterns []string) error {
	for f := range files {
		isSkipped := false
		for _, pattern := range skipPatterns {
			matched, err := filepath.Match(pattern, filepath.Base(f.path))
			if err != nil {
				return fmt.Errorf("error while trying to match file %s against glob pattern %s: %w", filepath.Base(f.path), pattern, err)
			}
			if matched {
				isSkipped = true
				break
			}
		}
		if isSkipped {
			delete(files, f)
		}
	}
	return nil
}

func materializeFiles(files map[*inMemoryFile]any) error {
	for f := range files {
		// create any intermediate directories required. Directories are lazily generated
		// only when they are required for a file.
		err := os.MkdirAll(filepath.Dir(f.path), 0755)
		if err != nil {
			return err
		}

		// write content to file
		err = os.WriteFile(f.path, []byte(f.content), f.perm)
		if err != nil {
			return err
		}
	}
	return nil
}

func walkFileTree(r *renderer, templateRoot, instanceRoot string) error {
	return filepath.WalkDir(templateRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip if current entry is not a directory
		if !d.IsDir() {
			return nil
		}

		// get relative path to the template file, This forms the template for the
		// path to the file
		relPathTemplate, err := filepath.Rel(templateRoot, path)
		if err != nil {
			return err
		}

		// Compute in memory representation for files in the current directory
		files, err := r.generateDir(path, filepath.Join(instanceRoot, relPathTemplate))
		if err != nil {
			return err
		}

		// Materialize these files onto the disk
		return materializeFiles(files)
	})
}
