package template

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type renderer struct {
	config       map[string]any
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

func (r *renderer) generateFile(pathTemplate, contentTemplate string, perm fs.FileMode) error {
	// compute file content
	fileContent, err := r.executeTemplate(contentTemplate)
	if errors.Is(err, errSkipThisFile) {
		// skip this file
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to compute file content for %s. %w", pathTemplate, err)
	}

	// compute the path for this file
	path, err := r.executeTemplate(pathTemplate)
	if err != nil {
		return fmt.Errorf("failed to compute path for %s. %w", pathTemplate, err)
	}
	// create any intermediate directories required. Directories are lazily generated
	// only when they are required for a file.
	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	// write content to file
	return os.WriteFile(path, []byte(fileContent), perm)
}

// TODO: use local filer client for this function. https://github.com/databricks/cli/issues/511
func walkFileTree(r *renderer, templateRoot, instanceRoot string) error {
	return filepath.WalkDir(templateRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip if current entry is a directory
		if d.IsDir() {
			return nil
		}

		// read template file to get the templatized content for the file
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		contentTemplate := string(b)

		// get relative path to the template file, This forms the template for the
		// path to the file
		relPathTemplate, err := filepath.Rel(templateRoot, path)
		if err != nil {
			return err
		}

		// Get info about the template file. Used to ensure instance path also
		// has the same permission bits
		info, err := d.Info()
		if err != nil {
			return err
		}

		return r.generateFile(filepath.Join(instanceRoot, relPathTemplate), contentTemplate, info.Mode().Perm())
	})
}
