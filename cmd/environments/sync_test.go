package environments

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupLocalConstraintsOnlyDeprecated pins --constraints-only as a hidden,
// deprecated alias for --no-dbconnect: the flag is kept defined so scripts and CI
// that already pass it keep working, but it is hidden from --help and pflag prints
// a one-line deprecation notice pointing at --no-dbconnect when it is used.
func TestSetupLocalConstraintsOnlyDeprecated(t *testing.T) {
	cmd := newSetupLocalCommand()

	f := cmd.Flags().Lookup("constraints-only")
	require.NotNil(t, f, "--constraints-only must remain defined for backward compatibility")
	assert.True(t, f.Hidden, "--constraints-only should be hidden from --help")
	assert.Equal(t, "use --no-dbconnect instead", f.Deprecated, "--constraints-only should carry the deprecation notice")
}
