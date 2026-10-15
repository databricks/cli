package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterFields(t *testing.T) {
	type volume struct {
		Comment string
		NewName string
		Owner   string
		Name    string
	}
	fields := []string{"Comment", "NewName", "Owner", "Name", "NotExistingField"}
	result := FilterFields[volume](fields, "NewName", "Owner")
	expected := []string{"Comment", "Name"}
	assert.Equal(t, expected, result)
}
