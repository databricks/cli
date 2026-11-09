package clusterpolicy

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
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

func TestDefinitionSupplies(t *testing.T) {
	definition, err := ParseDefinition(`{
		"spark_version": {"type": "fixed", "value": "15.4.x-scala2.12"},
		"autotermination_minutes": {"type": "range", "defaultValue": 60},
		"custom_tags.CostCenter": {"type": "fixed", "value": "dev-1234"},
		"spark_conf.spark.sql.shuffle.partitions": {"type": "unlimited", "defaultValue": 7},
		"spark_conf.*": {"type": "fixed", "value": "any"},
		"init_scripts.0.volumes.destination": {"type": "allowlist", "values": ["/Volumes/a"]},
		"init_scripts.*.volumes.destination": {"type": "fixed", "value": "/Volumes/shared"},
		"ssh_public_keys.*": {"type": "fixed", "value": "ssh-rsa key"},
		"cluster_log_conf.type": {"type": "fixed", "value": "DBFS"},
		"cluster_log_conf.path": {"type": "fixed", "value": "dbfs:/logs"},
		"cluster_log_conf.region": {"type": "fixed", "value": "us-west-2"}
	}`)
	require.NoError(t, err)

	for _, tt := range []struct {
		name          string
		path          string
		value         any
		applyDefaults bool
		want          bool
	}{
		{"fixed matches its value", "spark_version", "15.4.x-scala2.12", false, true},
		{"fixed matches a stale value", "spark_version", "14.3.x-scala2.12", false, true},
		{"default requires the flag", "autotermination_minutes", int64(60), false, false},
		{"default matches with the flag", "autotermination_minutes", int64(60), true, true},
		{"default requires equality", "autotermination_minutes", int64(30), true, false},
		{"unconstrained attribute", "node_type_id", "i3.xlarge", true, false},
		{"dotted map key is one attribute", "spark_conf['spark.sql.shuffle.partitions']", "7", true, true},
		{"map wildcard does not supply entries", "spark_conf['spark.other']", "any", true, false},
		{"map key in policy language", "custom_tags.CostCenter", "dev-1234", false, true},
		{"indexed rule masks the wildcard", "init_scripts[0].volumes.destination", "/Volumes/shared", false, false},
		{"wildcard applies to other indices", "init_scripts[1].volumes.destination", "/Volumes/other", false, true},
		{"wildcard array element", "ssh_public_keys[3]", "other", false, true},
		{"log destination matches type and path", "cluster_log_conf.dbfs.destination", "dbfs:/logs", false, true},
		{"log destination of another storage", "cluster_log_conf.s3.destination", "dbfs:/logs", false, false},
		{"log region alias", "cluster_log_conf.s3.region", "eu-west-1", false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, definition.Supplies(path, tt.value, tt.applyDefaults))
		})
	}
}
