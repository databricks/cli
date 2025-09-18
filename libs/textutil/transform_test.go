package textutil

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeMarks(t *testing.T) {
	x := NormalizeMarks()
	assert.Equal(t, "cafe", x.TransformString("café"))
	assert.Equal(t, "cafe 🍎", x.TransformString("café 🍎"))
	assert.Equal(t, "Foo Bar", x.TransformString("Foo Bar"))
}

func TestReplace(t *testing.T) {
	assert.Equal(t, "___abc___", ReplaceIn(unicode.Digit, '_').TransformString("000abc999"))
	assert.Equal(t, "___000___", ReplaceNotIn(unicode.Digit, '_').TransformString("abc000abc"))
}

func TestTrim(t *testing.T) {
	assert.Equal(t, "abc", TrimIfIn(unicode.Digit).TransformString("000abc999"))
	assert.Equal(t, "000", TrimIfNotIn(unicode.Digit).TransformString("abc000abc"))
}
