package annotation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileSetField(t *testing.T) {
	f := File{}

	// SetField allocates the type entry and its field map on first use.
	f.SetField("pkg.Type", "a", Descriptor{Description: "A"})
	f.SetField("pkg.Type", "b", Descriptor{Description: "B"})

	assert.Equal(t, "A", f["pkg.Type"].Fields["a"].Description)
	assert.Equal(t, "B", f["pkg.Type"].Fields["b"].Description)

	// A later SetField overwrites the field, leaving siblings intact.
	f.SetField("pkg.Type", "a", Descriptor{Description: "A2"})
	assert.Equal(t, "A2", f["pkg.Type"].Fields["a"].Description)
	assert.Equal(t, "B", f["pkg.Type"].Fields["b"].Description)
}

func TestFileSetSelf(t *testing.T) {
	f := File{}

	f.SetSelf("pkg.Type", Descriptor{Description: "the type"})
	assert.Equal(t, "the type", f["pkg.Type"].Self.Description)

	// SetSelf and SetField populate the same entry without clobbering each
	// other.
	f.SetField("pkg.Type", "a", Descriptor{Description: "A"})
	assert.Equal(t, "the type", f["pkg.Type"].Self.Description)
	assert.Equal(t, "A", f["pkg.Type"].Fields["a"].Description)
}
