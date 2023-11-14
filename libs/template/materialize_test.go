package template

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaterializeForNonTemplateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	err := Materialize(context.Background(), "", tmpDir, "")
	assert.EqualError(t, err, fmt.Sprintf("not a bundle template: expected to find a template schema file at %s", filepath.Join(tmpDir, schemaFileName)))
}
