package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLAnchors(t *testing.T) {
	b := load(t, "./yaml_anchors")
	assert.Len(t, b.Config.Resources.Jobs, 1)

	j := b.Config.Resources.Jobs["my_job"]
	require.Len(t, j.Tasks, 2)

	t0 := j.Tasks[0]
	t1 := j.Tasks[1]
	require.NotNil(t, t0)
	require.NotNil(t, t1)

	require.NotNil(t, t0.NewCluster)
	require.NotNil(t, t1.NewCluster)
	assert.Equal(t, "10.4.x-scala2.12", t0.NewCluster.SparkVersion)
	assert.Equal(t, "10.4.x-scala2.12", t1.NewCluster.SparkVersion)
}

func TestYAMLAnchorsNoWarnings(t *testing.T) {
	_, diags := loadTargetWithDiags("./yaml_anchors", "default")
	assert.Empty(t, diags)
}

func TestYAMLAnchorsSeparateBlockNoWarnings(t *testing.T) {
	_, diags := loadTargetWithDiags("./yaml_anchors_separate_block", "default")
	assert.Empty(t, diags)
}
