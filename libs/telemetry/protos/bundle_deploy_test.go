package protos

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBundleDeployEventOnlyHasNonPiiStringFields(t *testing.T) {
	allowedStringFields := map[string]bool{
		"BundleUuid":   true,
		"DeploymentId": true,
	}

	typ := reflect.TypeFor[BundleDeployEvent]()
	for field := range typ.Fields() {
		if field.Type.Kind() != reflect.String {
			continue
		}
		assert.True(t, allowedStringFields[field.Name],
			"%s may contain arbitrary free text; add an allow-listed non-PII field instead", field.Name)
	}
}
