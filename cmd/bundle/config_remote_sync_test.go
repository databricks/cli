package bundle

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/bundle/configsync"
	"github.com/stretchr/testify/require"
)

func TestRetrySourceChanges(t *testing.T) {
	attempts := 0
	err := retrySourceChanges(t.Context(), func() error {
		attempts++
		if attempts == 1 {
			return configsync.ErrSourceChanged
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, attempts)
}

func TestRetrySourceChangesStopsWhenRecoveryIsRequired(t *testing.T) {
	attempts := 0
	recoveryErr := errors.Join(configsync.ErrSourceChanged, configsync.ErrSourceRecoveryRequired)
	err := retrySourceChanges(t.Context(), func() error {
		attempts++
		return recoveryErr
	})
	require.ErrorIs(t, err, configsync.ErrSourceRecoveryRequired)
	require.Equal(t, 1, attempts)
}

func TestRetrySourceChangesStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := retrySourceChanges(ctx, func() error {
		return configsync.ErrSourceChanged
	})
	require.ErrorIs(t, err, context.Canceled)
}
