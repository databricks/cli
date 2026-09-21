package registry

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type single struct {
	Key   string `json:"key,omitempty"`
	Other string `json:"other,omitempty"`
}

type multi struct {
	UserName string `json:"user_name,omitempty"`
	SpName   string `json:"service_principal_name,omitempty"`
	Level    string `json:"level,omitempty"`
}

type embedInner struct {
	ID string `json:"id,omitempty"`
}

type embedOuter struct {
	embedInner
	Value string `json:"value,omitempty"`
}

// shadowOuter declares its own "id" and also embeds embedInner (which has "id"); the
// outer field is the one encoding/json serializes, so registration must select it.
type shadowOuter struct {
	embedInner
	ID string `json:"id,omitempty"`
}

func init() {
	Register[single]("key")
	Register[multi]("user_name", "service_principal_name")
	Register[embedOuter]("id")
	Register[shadowOuter]("id")
}

func TestKeyFields(t *testing.T) {
	assert.Equal(t, []string{"key"}, KeyFields(reflect.TypeFor[single]()))
	assert.Equal(t, []string{"user_name", "service_principal_name"}, KeyFields(reflect.TypeFor[multi]()))
	// Pointers are dereferenced.
	assert.Equal(t, []string{"key"}, KeyFields(reflect.TypeFor[*single]()))
	// Unregistered types return nil.
	assert.Nil(t, KeyFields(reflect.TypeFor[embedInner]()))
}

func TestElementKey(t *testing.T) {
	// First non-empty key field wins.
	v, ok := ElementKey(reflect.ValueOf(multi{SpName: "sp"}))
	assert.True(t, ok)
	assert.Equal(t, "sp", v)

	v, ok = ElementKey(reflect.ValueOf(multi{UserName: "u", SpName: "sp"}))
	assert.True(t, ok)
	assert.Equal(t, "u", v)

	// Registered but all key fields empty: ok, empty value.
	v, ok = ElementKey(reflect.ValueOf(multi{Level: "CAN_MANAGE"}))
	assert.True(t, ok)
	assert.Empty(t, v)

	// Key resolved through a flattened embed.
	v, ok = ElementKey(reflect.ValueOf(embedOuter{embedInner: embedInner{ID: "x"}}))
	assert.True(t, ok)
	assert.Equal(t, "x", v)

	// Unregistered type.
	_, ok = ElementKey(reflect.ValueOf(embedInner{ID: "x"}))
	assert.False(t, ok)
}

func TestElementKeyShadowedEmbed(t *testing.T) {
	// The outer "id" shadows the embedded one; ElementKey reads the outer field.
	v, ok := ElementKey(reflect.ValueOf(shadowOuter{ID: "outer", embedInner: embedInner{ID: "inner"}}))
	assert.True(t, ok)
	assert.Equal(t, "outer", v)
}

func TestElementKeyNilPointer(t *testing.T) {
	// A nil pointer element must not panic.
	v, ok := ElementKey(reflect.ValueOf((*multi)(nil)))
	assert.False(t, ok)
	assert.Empty(t, v)
}

func TestRegisterPanicsOnUnknownField(t *testing.T) {
	assert.PanicsWithValue(t,
		`registry: registry.single has no string field with json name "missing"`,
		func() { Register[single]("missing") },
	)
}
