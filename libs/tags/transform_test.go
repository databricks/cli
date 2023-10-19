package tags

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeMarks(t *testing.T) {
	x := normalizeMarks()
	assert.Equal(t, "cafe", x.transform("café"))
	assert.Equal(t, "cafe 🍎", x.transform("café 🍎"))
	assert.Equal(t, "Foo Bar", x.transform("Foo Bar"))
}

func TestReplace(t *testing.T) {
	assert.Equal(t, "___abc___", replaceIn(unicode.Digit, '_').transform("000abc999"))
	assert.Equal(t, "___000___", replaceNotIn(unicode.Digit, '_').transform("abc000abc"))
}

func TestTrim(t *testing.T) {
	assert.Equal(t, "abc", trimIfIn(unicode.Digit).transform("000abc999"))
	assert.Equal(t, "000", trimIfNotIn(unicode.Digit).transform("abc000abc"))
}
