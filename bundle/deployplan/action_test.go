package deployplan

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringShort(t *testing.T) {
	shortMap := make(map[string][]string)

	for a, s := range actionName {
		require.Equal(t, a.StringFull(), s)
		short := a.StringShort()
		require.NotEmpty(t, short)
		require.True(t, strings.HasPrefix(s, short), "%q %q", s, short)
		if short != s {
			shortMap[short] = append(shortMap[short], s)
		}
	}

	require.Equal(t, map[string][]string{
		"update": {
			"update(id_stable)",
			"update(id_changes)",
		},
	}, shortMap)
}
