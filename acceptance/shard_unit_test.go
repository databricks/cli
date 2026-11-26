package acceptance_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShardEveryTestInOneShard(t *testing.T) {
	const n = 6
	for i := range 200 {
		dir := fmt.Sprintf("bundle/test%d", i)
		envset := []string{fmt.Sprintf("DMS=%d", i%2)}
		runs := 0
		for shard := range n {
			if (shardSelector{index: shard, count: n}).skipReason(dir, envset) == "" {
				runs++
			}
		}
		assert.Equal(t, 1, runs, dir)
	}
}

func TestShardDisabled(t *testing.T) {
	assert.Empty(t, shardSelector{}.skipReason("bundle/foo", nil))
}

func TestParseShard(t *testing.T) {
	s, err := parseShard("2/6")
	require.NoError(t, err)
	assert.Equal(t, shardSelector{index: 2, count: 6}, s)

	for _, raw := range []string{"6/6", "-1/6", "1/0", "1", "a/6"} {
		_, err := parseShard(raw)
		assert.Error(t, err, raw)
	}
}
