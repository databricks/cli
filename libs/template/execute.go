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
func generateDirectory(config map[string]any, parentDir, nameTempate string) (string, error) {
	dirName, err := executeTemplate(config, nameTempate)
	if err != nil {
		return "", err
	}
	err = os.Mkdir(filepath.Join(parentDir, dirName), 0755)
	if err != nil {
		return "", err
	}
	return dirName, nil
}

func generateFile(config map[string]any, parentDir, nameTempate, contentTemplate string) error {
	// compute file content
	fileContent, err := executeTemplate(config, contentTemplate)
	if errors.Is(err, errSkipThisFile) {
		return nil
	}
	if err != nil {
		return err
	}

	// create the file by executing the templatized file name
	fileName, err := executeTemplate(config, nameTempate)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(parentDir, fileName), []byte(fileContent), 0644)
}

func WalkFileTree(config map[string]any, templatePath, instancePath string) error {
	entries, err := os.ReadDir(templatePath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// case: materialize a template directory
			dirName, err := generateDirectory(config, instancePath, entry.Name())
			if err != nil {
				return err
			}

			// recusive generate files and directories inside inside our newly generated
			// directory from the template defination
			err = WalkFileTree(config, filepath.Join(templatePath, entry.Name()), filepath.Join(instancePath, dirName))
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
			err = generateFile(config, instancePath, fileNameTemplate, contentTemplate)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
