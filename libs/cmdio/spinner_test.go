package cmdio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpinnerModelInit(t *testing.T) {
	m := newSpinnerModel()
	assert.False(t, m.quitting)
	assert.Equal(t, "", m.suffix)
	assert.NotNil(t, m.spinner)
}

func TestSpinnerModelUpdateSuffixMsg(t *testing.T) {
	m := newSpinnerModel()
	msg := suffixMsg("processing files")

	updatedModel, _ := m.Update(msg)
	updated := updatedModel.(spinnerModel)

	assert.Equal(t, "processing files", updated.suffix)
	assert.False(t, updated.quitting)
}

func TestSpinnerModelUpdateQuitMsg(t *testing.T) {
	m := newSpinnerModel()
	msg := quitMsg{}

	updatedModel, cmd := m.Update(msg)
	updated := updatedModel.(spinnerModel)

	assert.True(t, updated.quitting)
	assert.NotNil(t, cmd) // Should return tea.Quit
}

func TestSpinnerModelViewActive(t *testing.T) {
	m := newSpinnerModel()
	m.suffix = "loading"

	view := m.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "loading")
}

func TestSpinnerModelViewQuitting(t *testing.T) {
	m := newSpinnerModel()
	m.quitting = true

	view := m.View()

	assert.Empty(t, view)
}
