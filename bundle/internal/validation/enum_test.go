package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractEnumFieldsExcludesCatalogPrivilege(t *testing.T) {
	patterns, err := extractEnumFields(reflect.TypeFor[config.Root]())
	require.NoError(t, err)

	for _, pattern := range patterns {
		assert.NotContains(t, pattern.Pattern, ".grants[*].privileges[*]")
	}

	var hasUnrelated bool
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern.Pattern, ".permissions[*].level") {
			hasUnrelated = true
			break
		}
	}
	assert.True(t, hasUnrelated, "expected unrelated enum validation to remain")
}
