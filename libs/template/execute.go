package template

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Executes the template by applying config on it. Returns the materialized config
// as a string
// TODO: test this function
func executeTemplate(config map[string]any, templateDefinition string) (string, error) {
	// configure template with helper functions
	tmpl, err := template.New("").Funcs(HelperFuncs).Parse(templateDefinition)
	if err != nil {
		return "", err
	}

	// execute template
	result := strings.Builder{}
	err = tmpl.Execute(&result, config)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

// TODO: test this function
func generateFile(config map[string]any, pathTemplate, contentTemplate string, perm fs.FileMode) error {
	// compute file content
	fileContent, err := executeTemplate(config, contentTemplate)
	if errors.Is(err, errSkipThisFile) {
		// skip this file
		return nil
	}
	if err != nil {
		return err
	}

	// compute the path for this file
	path, err := executeTemplate(config, pathTemplate)
	if err != nil {
		return err
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

func walkFileTree(config map[string]any, templateRoot, instanceRoot string) error {
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

		return generateFile(config, filepath.Join(instanceRoot, relPathTemplate), contentTemplate, info.Mode().Perm())
	})
}
