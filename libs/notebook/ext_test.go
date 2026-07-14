package notebook

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
)

func TestStripExtension(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"foo.py", "foo"},
		{"foo.ipynb", "foo"},
		{"foo.r", "foo"},
		{"foo.scala", "foo"},
		{"foo.sql", "foo"},
		{"a/b/c.ipynb", "a/b/c"},

		// Designer files keep their full ".designer.ipynb" suffix.
		{"foo.designer.ipynb", "foo.designer.ipynb"},
		{"a/b/c.designer.ipynb", "a/b/c.designer.ipynb"},

		// Files without a known extension are passed through path.Ext;
		// the last-segment extension is removed.
		{"foo", "foo"},
		{"foo.unknown", "foo"},
	} {
		assert.Equal(t, tc.want, StripExtension(tc.in), "input=%q", tc.in)
	}
}

func TestFixedExportFormat(t *testing.T) {
	// Designer files report no export format and must round-trip as Jupyter.
	format, ok := FixedExportFormat(ObjectTypeDesignerFile)
	assert.True(t, ok)
	assert.Equal(t, workspace.ExportFormatJupyter, format)

	// Regular notebooks report their own export format.
	_, ok = FixedExportFormat(workspace.ObjectTypeNotebook)
	assert.False(t, ok)
}
