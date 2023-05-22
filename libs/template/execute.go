package template

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Executes the template by applying config on it. Returns the materialized config
// as a string
func executeTemplate(config map[string]any, templateDefination string) (string, error) {
	// configure template with helper functions
	tmpl, err := template.New("").Funcs(HelperFuncs).Parse(templateDefination)
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

// TODO: allow skipping directories

func generateFile(config map[string]any, pathTemplate, contentTemplate string) error {
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
	return os.WriteFile(path, []byte(fileContent), 0644)
}

func walkFileTree(config map[string]any, templatePath, instancePath string) error {
	entries, err := os.ReadDir(templatePath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// compute directory name
			dirName, err := executeTemplate(config, entry.Name())
			if err != nil {
				return err
			}

			// recusively generate files and directories inside inside our newly generated
			// directory from the template defination
			err = walkFileTree(config, filepath.Join(templatePath, entry.Name()), filepath.Join(instancePath, dirName))
			if err != nil {
				return err
			}
		} else {
			// case: materialize a template file with it's contents
			b, err := os.ReadFile(filepath.Join(templatePath, entry.Name()))
			if err != nil {
				return err
			}
			contentTemplate := string(b)
			fileNameTemplate := entry.Name()
			err = generateFile(config, filepath.Join(instancePath, fileNameTemplate), contentTemplate)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
