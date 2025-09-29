package deployplan

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringShort(t *testing.T) {
	shortMap := make(map[string][]string)

	keys := slices.Collect(maps.Keys(actionName))
	slices.Sort(keys)

	for _, a := range keys {
		s := actionName[a]
		require.Equal(t, a.String(), s)
		short := a.StringShort()
		require.NotEmpty(t, short)
		require.True(t, strings.HasPrefix(s, short), "%q %q", s, short)
		if short != s {
			shortMap[short] = append(shortMap[short], s)
		}
	}

	require.Equal(t, map[string][]string{
		"update": {
			"update_id",
		},
	}, shortMap)
}
