package template

import (
	"context"
	"path/filepath"
)

const libraryDirName = "library"
const templateDirName = "template"

func Materialize(ctx context.Context, config map[string]any, templateRoot, instanceRoot string) error {
	templatePath := filepath.Join(templateRoot, templateDirName)
	libraryPath := filepath.Join(templateRoot, libraryDirName)

	r, err := newRenderer(ctx, config, templatePath, libraryPath, instanceRoot)
	if err != nil {
		return err
	}
	err = r.walk()
	if err != nil {
		return err
	}
	return r.persistToDisk()
}
