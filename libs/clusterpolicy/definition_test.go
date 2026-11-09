package clusterpolicy

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDefinitionKeepsNumbers(t *testing.T) {
	definition, err := ParseDefinition(`{
		"autotermination_minutes": {"type": "fixed", "value": 60},
		"spark_conf.spark.sql.shuffle.partitions": {"type": "unlimited", "defaultValue": 7},
		"custom_tags.Team": {"type": "forbidden"}
	}`)
	require.NoError(t, err)
	assert.Equal(t, json.Number("60"), definition["autotermination_minutes"].EffectiveValue(false))
	assert.Nil(t, definition["spark_conf.spark.sql.shuffle.partitions"].EffectiveValue(false))
	assert.Equal(t, json.Number("7"), definition["spark_conf.spark.sql.shuffle.partitions"].EffectiveValue(true))
	assert.Nil(t, definition["custom_tags.Team"].EffectiveValue(true))
	assert.Nil(t, definition["missing"].EffectiveValue(true))

	_, err = ParseDefinition(`{"spark_version": `)
	assert.Error(t, err)
}

func TestValueEquals(t *testing.T) {
	for _, tt := range []struct {
		name         string
		policy, want any
		equal        bool
	}{
		{"int field", json.Number("60"), int64(60), true},
		{"float field", json.Number("1.5"), 1.5, true},
		{"string field holds number literal", json.Number("7"), "7", true},
		{"string field holds bool literal", true, "true", true},
		{"different number", json.Number("60"), int64(61), false},
		{"number against bool", json.Number("1"), true, false},
		{"strings", "dev", "dev", true},
		{"bools", false, false, true},
		{"nil policy value", nil, "dev", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, ValueEquals(tt.policy, tt.want))
		})
	}
}
