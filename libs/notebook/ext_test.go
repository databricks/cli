package notebook

import (
	"testing"

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

		// Flow files keep their full ".flow.ipynb" suffix.
		{"foo.flow.ipynb", "foo.flow.ipynb"},
		{"a/b/c.flow.ipynb", "a/b/c.flow.ipynb"},

		// Files without a known extension are passed through path.Ext;
		// the last-segment extension is removed.
		{"foo", "foo"},
		{"foo.unknown", "foo"},
	} {
		assert.Equal(t, tc.want, StripExtension(tc.in), "input=%q", tc.in)
	}
}
