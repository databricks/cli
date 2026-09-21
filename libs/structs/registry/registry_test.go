package registry

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type single struct {
	Key string `json:"key,omitempty"`
}

type multi struct {
	UserName string `json:"user_name,omitempty"`
	SpName   string `json:"service_principal_name,omitempty"`
}

type ptrKeyed struct {
	Name string `json:"pk_name,omitempty"`
}

func init() {
	Register[single]("key")
	Register[multi]("user_name", "service_principal_name")
	Register[*ptrKeyed]("pk_name")
}

func TestKeyFields(t *testing.T) {
	assert.Equal(t, []string{"key"}, KeyFields(reflect.TypeFor[single]()))
	assert.Equal(t, []string{"user_name", "service_principal_name"}, KeyFields(reflect.TypeFor[multi]()))
	// Unregistered types return nil.
	assert.Nil(t, KeyFields(reflect.TypeFor[struct{ X string }]()))
}

func TestKeyFieldsPointerNormalization(t *testing.T) {
	// Register[*T] normalizes to T, and lookups by either T or *T agree.
	assert.Equal(t, []string{"pk_name"}, KeyFields(reflect.TypeFor[ptrKeyed]()))
	assert.Equal(t, []string{"pk_name"}, KeyFields(reflect.TypeFor[*ptrKeyed]()))
	// A registered value type is also found through a pointer.
	assert.Equal(t, []string{"key"}, KeyFields(reflect.TypeFor[*single]()))
}
