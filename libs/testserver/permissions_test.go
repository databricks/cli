package testserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterPermissionsObjectTypeTwicePanics(t *testing.T) {
	assert.PanicsWithValue(t, "permissions object type registered twice: jobs", func() {
		registerPermissionsObjectType("jobs", "job")
	})
}
