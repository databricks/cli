package patchwheel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterLatestWheels(t *testing.T) {
	paths := []string{
		"mypkg-0.1.0-py3-none-any.whl",
		"mypkg-0.2.0-py3-none-any.whl",
		"other-1.0.0-py3-none-any.whl",
		"other-0.9.0-py3-none-any.whl",
	}

	filtered := FilterLatestWheels(context.Background(), paths)
	require.ElementsMatch(t, []string{
		"mypkg-0.2.0-py3-none-any.whl",
		"other-1.0.0-py3-none-any.whl",
	}, filtered)
}

func TestFilterLatestWheelsWithTimestamp(t *testing.T) {
	// Second package has timestamp suffix which should win over plain version.
	paths := []string{
		"mypkg-1.2.3-py3-none-any.whl",
		"mypkg-1.2.3+1741091696780123321-py3-none-any.whl",
	}
	filtered := FilterLatestWheels(context.Background(), paths)
	require.Equal(t, []string{"mypkg-1.2.3+1741091696780123321-py3-none-any.whl"}, filtered)
}

func TestFilterLatestWheelsKeepsUnparsable(t *testing.T) {
	paths := []string{
		"not-a-wheel.txt",
		"mypkg-0.1.0-py3-none-any.whl",
		"mypkg-0.2.0-py3-none-any.whl",
	}

	filtered := FilterLatestWheels(context.Background(), paths)
	require.Equal(t, []string{
		"not-a-wheel.txt",
		"mypkg-0.2.0-py3-none-any.whl",
	}, filtered)
}
