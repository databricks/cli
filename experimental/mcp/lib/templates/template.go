package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

// Template represents a project template
type Template interface {
	Name() string
	Description() string
	Files() (map[string]string, error)
}

// EmbeddedTemplate is a template loaded from an embedded filesystem
type EmbeddedTemplate struct {
	name        string
	description string
	fsys        embed.FS
	root        string
}

// NewEmbeddedTemplate creates a new embedded template
func NewEmbeddedTemplate(name, desc string, fsys embed.FS, root string) *EmbeddedTemplate {
	return &EmbeddedTemplate{
		name:        name,
		description: desc,
		fsys:        fsys,
		root:        root,
	}
}

// Name returns the template name
func (t *EmbeddedTemplate) Name() string {
	return t.name
}

// Description returns the template description
func (t *EmbeddedTemplate) Description() string {
	return t.description
}

// Files returns a map of file paths to their contents
func (t *EmbeddedTemplate) Files() (map[string]string, error) {
	files := make(map[string]string)

	err := fs.WalkDir(t.fsys, t.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Read file content
		content, err := fs.ReadFile(t.fsys, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Remove root prefix from path
		relativePath := strings.TrimPrefix(path, t.root+"/")
		if relativePath == "" {
			relativePath = path
		}
		files[relativePath] = string(content)

		return nil
	})

	return files, err
}
