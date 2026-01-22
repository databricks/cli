package dsc

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name    string
		fields  []RequiredField
		wantErr bool
		errMsg  string
	}{
		{
			name: "all fields present",
			fields: []RequiredField{
				{Name: "scope", Value: "my-scope"},
				{Name: "key", Value: "my-key"},
			},
			wantErr: false,
		},
		{
			name: "first field missing",
			fields: []RequiredField{
				{Name: "scope", Value: ""},
				{Name: "key", Value: "my-key"},
			},
			wantErr: true,
			errMsg:  "scope is required",
		},
		{
			name: "second field missing",
			fields: []RequiredField{
				{Name: "scope", Value: "my-scope"},
				{Name: "key", Value: ""},
			},
			wantErr: true,
			errMsg:  "key is required",
		},
		{
			name:    "no fields",
			fields:  []RequiredField{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequired(tt.fields...)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateAtLeastOne(t *testing.T) {
	tests := []struct {
		name        string
		description string
		values      []string
		wantErr     bool
	}{
		{
			name:        "first value present",
			description: "string_value or bytes_value",
			values:      []string{"hello", ""},
			wantErr:     false,
		},
		{
			name:        "second value present",
			description: "string_value or bytes_value",
			values:      []string{"", "bytes"},
			wantErr:     false,
		},
		{
			name:        "both values present",
			description: "string_value or bytes_value",
			values:      []string{"hello", "bytes"},
			wantErr:     false,
		},
		{
			name:        "no values present",
			description: "string_value or bytes_value",
			values:      []string{"", ""},
			wantErr:     true,
		},
		{
			name:        "single empty value",
			description: "value",
			values:      []string{""},
			wantErr:     true,
		},
		{
			name:        "single present value",
			description: "value",
			values:      []string{"present"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAtLeastOne(tt.description, tt.values...)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.description)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUnmarshalInput(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name    string
		input   string
		want    testStruct
		wantErr bool
	}{
		{
			name:  "valid JSON",
			input: `{"name": "test", "value": 42}`,
			want:  testStruct{Name: "test", Value: 42},
		},
		{
			name:  "partial JSON",
			input: `{"name": "test"}`,
			want:  testStruct{Name: "test", Value: 0},
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:  "empty object",
			input: `{}`,
			want:  testStruct{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unmarshalInput[testStruct]([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestNotFoundError(t *testing.T) {
	tests := []struct {
		name        string
		resourceTyp string
		identifiers []string
		wantContain string
	}{
		{
			name:        "single identifier",
			resourceTyp: "secret",
			identifiers: []string{"my-key"},
			wantContain: "secret not found",
		},
		{
			name:        "multiple identifiers",
			resourceTyp: "secret",
			identifiers: []string{"scope=my-scope", "key=my-key"},
			wantContain: "secret not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NotFoundError(tt.resourceTyp, tt.identifiers...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantContain)
		})
	}
}

func TestDefaultExitCodes(t *testing.T) {
	codes := DefaultExitCodes()
	assert.Equal(t, "Success", codes["0"])
	assert.Equal(t, "Error", codes["1"])
	assert.Len(t, codes, 2)
}
