package cmdio

import (
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/stretchr/testify/assert"
)

func TestWriterPassesThroughWhenColorOn(t *testing.T) {
	r := &Renderer{profile: colorprofile.ANSI}
	styled := "\x1b[38;2;143;179;220mlabel\x1b[m " + Hyperlink("https://h.test", "docs")

	var b strings.Builder
	_, err := r.Writer(&b).Write([]byte(styled))
	assert.NoError(t, err)
	assert.Equal(t, styled, b.String())
}
