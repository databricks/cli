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
	assert.EqualError(t, err, fmt.Sprintf("expected to find a template schema file at %s. Valid bundle templates are expected to contain a schema file", filepath.Join(tmpDir, schemaFileName)))
}
