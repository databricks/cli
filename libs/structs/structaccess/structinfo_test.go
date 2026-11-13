package structaccess

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type InfoInner struct {
	A               string   `json:"a,omitempty"`
	ForceSendFields []string `json:"-"`
}

type InfoTagged struct {
	B string `json:"b"`
}

type infoOuter struct {
	InfoInner
	InfoTagged      `json:"tagged"`
	Named           string `json:"named,omitempty"`
	NoTag           int
	Skipped         string   `json:"-"`
	Dash            string   `json:"-,omitempty"` //nolint:staticcheck // the field is named "-"
	EmbeddedSlice   []string `json:"items"`
	unexported      string   //nolint:unused
	ForceSendFields []string `json:"-"`
}

type infoPromoted struct {
	InfoInner
	Own string `json:"own,omitempty"`
}

func TestGetStructInfo(t *testing.T) {
	info := GetStructInfo(reflect.TypeFor[infoOuter]())

	type summary struct {
		Index         int
		Name          string
		OmitEmpty     bool
		Flattened     bool
		EmbeddedSlice bool
	}
	var got []summary
	for _, f := range info.Fields {
		got = append(got, summary{f.Index, f.Name, f.OmitEmpty, f.Flattened, f.EmbeddedSlice})
	}
	assert.Equal(t, []summary{
		{Index: 0, Flattened: true},
		{Index: 1, Name: "tagged"},
		{Index: 2, Name: "named", OmitEmpty: true},
		{Index: 3, Name: "NoTag"},
		{Index: 5, Name: "-", OmitEmpty: true},
		{Index: 6, Name: "items", EmbeddedSlice: true},
	}, got)
	assert.Equal(t, 8, info.ForceSendFieldsIndex)

	v := reflect.ValueOf(infoOuter{ForceSendFields: []string{"Named"}})
	assert.Equal(t, []string{"Named"}, info.ForceSendFields(v))

	assert.Same(t, info, GetStructInfo(reflect.TypeFor[infoOuter]()))
}

// A ForceSendFields promoted from an embedded struct governs only that struct's fields.
func TestGetStructInfoPromotedForceSendFields(t *testing.T) {
	info := GetStructInfo(reflect.TypeFor[infoPromoted]())
	assert.Equal(t, -1, info.ForceSendFieldsIndex)

	v := reflect.ValueOf(infoPromoted{ForceSendFields: []string{"Own"}})
	assert.Nil(t, info.ForceSendFields(v))
}
