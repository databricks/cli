package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateMaterializedConfigIncludesRecordRequests(t *testing.T) {
	trueValue := true
	falseValue := false
	assert.Equal(t, "RecordRequests = true\n", GenerateMaterializedConfig(&TestConfig{RecordRequests: &trueValue}))
	assert.Equal(t, "RecordRequests = false\n", GenerateMaterializedConfig(&TestConfig{RecordRequests: &falseValue}))
	assert.Equal(t, "RecordRequests = false\n", GenerateMaterializedConfig(&TestConfig{}))
}
