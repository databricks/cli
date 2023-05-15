package init

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TODO: cleanup if template initialization fails
// TODO: need robust way to clean up half generated files
// TODO: define default files
// TODO: self reference for 

func walkFileTree(config map[string]interface{}, templatePath string, instancePath string) error {
	enteries, err := os.ReadDir(templatePath)
	if err != nil {
		return err
	}
	for _, entry := range enteries {
		if entry.Name() == SchemaFileName {
			continue
		}
		fileName := entry.Name()
		tmpl, err := template.New("filename").Parse(fileName)
		if err != nil {
			return err
		}
		result := strings.Builder{}
		err = tmpl.Execute(&result, config)
		if err != nil {
			return err
		}
		resolvedFileName := result.String()
		fmt.Println(resolvedFileName)
		if entry.IsDir() {
			err := os.Mkdir(resolvedFileName, os.ModePerm)
			if err != nil {
				return err
			}
			err = walkFileTree(config, filepath.Join(templatePath, fileName), filepath.Join(instancePath, resolvedFileName))
			if err != nil {
				return err
			}
		} else {
			f, err := os.Create(filepath.Join(instancePath, resolvedFileName))
			if err != nil {
				return err
			}
			b, err := os.ReadFile(filepath.Join(templatePath, fileName))
			if err != nil {
				return err
			}
			// TODO: Might be able to use ParseFiles or ParseFS. Might be more suited
			contentTmpl, err := template.New("content").Funcs(HelperFuncs).Parse(string(b))
			if err != nil {
				return err
			}
			err = contentTmpl.Execute(f, config)

			// Make this assertion more robust
			if err != nil && strings.Contains(err.Error(), ErrSkipThisFile.Error()) {
				err := os.Remove(filepath.Join(instancePath, resolvedFileName))
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}
	}
	return nil
}
