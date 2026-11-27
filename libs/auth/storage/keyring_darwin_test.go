package storage

import (
	"os"
	"strings"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// TestZalandoBackend_Set_RejectsLargeSecretWithoutSecurityProcess pins that
// zalandoBackend rejects a secret that is too big on macOS without calling
// go-keyring, which would start `security -i` and leave its stdin pipe open in
// this process. Nothing reaches the keychain either way, so it runs outside CI
// too. This file sorts before keyring_size_limit_os_test.go, so the test runs
// before TestKeyringSet_RejectsLargeSecret, which calls go-keyring directly and
// leaves such a pipe, which a finalizer could close between the two counts.
func TestZalandoBackend_Set_RejectsLargeSecretWithoutSecurityProcess(t *testing.T) {
	before := openFileCount(t)

	err := zalandoBackend{}.Set(keyringServiceName, keyringProbeAccountPrefix+uuid.New().String(), strings.Repeat("a", 5000))
	require.ErrorIs(t, err, keyring.ErrSetDataTooBig)
	assert.Equal(t, before, openFileCount(t), "go-keyring started `security -i`, which keeps a pipe open")
}

// openFileCount returns the number of files this process has open.
func openFileCount(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/dev/fd")
	require.NoError(t, err)
	return len(entries)
}
