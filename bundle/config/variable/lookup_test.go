package variable

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookup_Coverage(t *testing.T) {
	var lookup Lookup
	val := reflect.ValueOf(lookup)
	typ := val.Type()

	for i := range val.NumField() {
		field := val.Field(i)
		if field.Kind() != reflect.String {
			t.Fatalf("Field %s is not a string", typ.Field(i).Name)
		}

		fieldType := typ.Field(i)
		t.Run(fieldType.Name, func(t *testing.T) {
			// Use a fresh instance of the struct in each test
			var lookup Lookup

			// Set the field to a non-empty string
			reflect.ValueOf(&lookup).Elem().Field(i).SetString("value")

			// Test the [String] function
			assert.NotEmpty(t, lookup.String())
		})
	}
}

func TestLookup_Empty(t *testing.T) {
	var lookup Lookup

	// Resolve returns an error when no fields are provided
	_, err := lookup.Resolve(context.Background(), nil)
	assert.ErrorContains(t, err, "no valid lookup fields provided")

	// No string representation for an invalid lookup
	assert.Empty(t, lookup.String())
}

func TestLookup_Multiple(t *testing.T) {
	lookup := Lookup{
		Alert: "alert",
		Query: "query",
	}

	// Resolve returns an error when multiple fields are provided
	_, err := lookup.Resolve(context.Background(), nil)
	assert.ErrorContains(t, err, "exactly one lookup field must be provided")

	// No string representation for an invalid lookup
	assert.Empty(t, lookup.String())
}
