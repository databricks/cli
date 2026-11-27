package autoinstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMissingFileIsZeroState(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())

	s, err := Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, State{}, s)
}

func TestLoadStateWithoutLastAttempt(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	path := filepath.Join(home, ".databricks", "aitools", stateFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	// The format of CLI versions before last_attempt was added.
	require.NoError(t, os.WriteFile(path, []byte(`{"last_version": "1.27.0", "attempts": 3, "id": "machine"}`), 0o600))

	s, err := Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, State{LastVersion: "1.27.0", Attempts: 3, ID: "machine"}, s)
}

func TestLoadCorruptFileFails(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	path := filepath.Join(home, ".databricks", "aitools", stateFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	_, err := Load(ctx)
	assert.ErrorContains(t, err, "parse ")
}

func TestLoadNegativeAttemptsFails(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	path := filepath.Join(home, ".databricks", "aitools", stateFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`{"last_version": "1.30.0", "attempts": -1}`), 0o600))

	_, err := Load(ctx)
	assert.ErrorContains(t, err, "negative attempts -1")
}

// writeStateFile writes data as the state file of home.
func writeStateFile(t *testing.T, home, data string) {
	t.Helper()
	path := filepath.Join(home, ".databricks", "aitools", stateFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
}

// State files as CLI 1.27 and 1.28 write them, which know only opted_out.
func TestLoadStateOfEarlierVersions(t *testing.T) {
	at := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		data string
		want State
	}{
		{
			name: "1.27 opted out",
			data: `{"last_version": "1.27.0", "opted_out": true, "id": "machine"}`,
			want: State{LastVersion: "1.27.0", OptedOut: true, ID: "machine"},
		},
		{
			name: "1.27 after a yes at the install prompt",
			data: `{"last_version": "1.27.0", "accepted_version": "1.27.0", "installing": true, "id": "machine"}`,
			want: State{LastVersion: "1.27.0", AcceptedVersion: "1.27.0", Installing: true, ID: "machine"},
		},
		{
			name: "1.28 opted out during retries",
			data: `{"last_version": "1.28.0", "attempts": 2, "last_attempt": "2026-11-20T10:00:00Z", "opted_out": true, "id": "machine"}`,
			want: State{LastVersion: "1.28.0", Attempts: 2, LastAttempt: at, OptedOut: true, ID: "machine"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			ctx := env.WithUserHomeDir(t.Context(), home)
			writeStateFile(t, home, tc.data)

			s, err := Load(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, s)
			want := ConsentUnknown
			if tc.want.OptedOut {
				want = ConsentOptedOut
			}
			assert.Equal(t, want, s.Consent())
		})
	}
}

// olderState is the state as CLI 1.28 reads it.
type olderState struct {
	LastVersion     string    `json:"last_version,omitempty"`
	Attempts        int       `json:"attempts,omitempty"`
	LastAttempt     time.Time `json:"last_attempt,omitzero"`
	OptedOut        bool      `json:"opted_out,omitempty"`
	AcceptedVersion string    `json:"accepted_version,omitempty"`
	Installing      bool      `json:"installing,omitempty"`
	ID              string    `json:"id,omitempty"`
}

// A CLI from before the consent was stored reads the file that this one
// writes: an opt-out is one for it too, and a consent is not.
func TestOlderCLIReadsTheConsent(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	path := filepath.Join(home, ".databricks", "aitools", stateFileName)
	readOlder := func() olderState {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var s olderState
		require.NoError(t, json.Unmarshal(data, &s))
		return s
	}

	_, err := SetConsent(ctx, ConsentGiven)
	require.NoError(t, err)
	assert.False(t, readOlder().OptedOut)

	_, err = SetConsent(ctx, ConsentOptedOut)
	require.NoError(t, err)
	assert.True(t, readOlder().OptedOut)
}

func TestConsent(t *testing.T) {
	assert.Equal(t, ConsentUnknown, State{}.Consent())
	assert.Equal(t, ConsentGiven, State{Consented: true}.Consent())
	assert.Equal(t, ConsentOptedOut, State{OptedOut: true}.Consent())
	assert.Equal(t, ConsentOptedOut, State{OptedOut: true, Consented: true}.Consent(), "a hand-edited file with both is opted out")
}

func TestSetConsentKeepsOtherFields(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	require.NoError(t, save(ctx, State{LastVersion: "1.2.3", Attempts: 2}))

	for _, step := range []struct {
		consent      Consent
		wantPrevious Consent
		want         State
	}{
		{ConsentOptedOut, ConsentUnknown, State{LastVersion: "1.2.3", Attempts: 2, OptedOut: true}},
		{ConsentGiven, ConsentOptedOut, State{LastVersion: "1.2.3", Attempts: 2, Consented: true}},
		{ConsentOptedOut, ConsentGiven, State{LastVersion: "1.2.3", Attempts: 2, OptedOut: true}},
	} {
		previous, err := SetConsent(ctx, step.consent)
		require.NoError(t, err)
		assert.Equal(t, step.wantPrevious, previous)
		s, err := Load(ctx)
		require.NoError(t, err)
		assert.Equal(t, step.want, s)
	}
}

func TestSetConsentReplacesCorruptFile(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	writeStateFile(t, home, "not json")

	previous, err := SetConsent(ctx, ConsentOptedOut)
	require.NoError(t, err)
	assert.Equal(t, ConsentUnknown, previous)
	s, err := Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, State{OptedOut: true}, s)
}

func TestGiveConsentIfUnknown(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state State
		want  State
	}{
		{"unknown", State{LastVersion: "1.2.3"}, State{LastVersion: "1.2.3", Consented: true}},
		{"unknown after an install without consent", State{InstalledWithoutConsent: true}, State{Consented: true}},
		{"given", State{Consented: true}, State{Consented: true}},
		{"an opt-out stays", State{OptedOut: true}, State{OptedOut: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
			require.NoError(t, save(ctx, tc.state))

			previous, err := GiveConsentIfUnknown(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.state.Consent(), previous)
			s, err := Load(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, s)
		})
	}
}

// The unreadable file may hold an opt-out, which an implied consent must not
// replace.
func TestGiveConsentIfUnknownKeepsACorruptFile(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	writeStateFile(t, home, "not json")

	_, err := GiveConsentIfUnknown(ctx)
	assert.ErrorContains(t, err, "parse ")
	data, err := os.ReadFile(filepath.Join(home, ".databricks", "aitools", stateFileName))
	require.NoError(t, err)
	assert.Equal(t, "not json", string(data))
}

func TestEnvOptedOut(t *testing.T) {
	ctx := t.Context()
	assert.False(t, EnvOptedOut(ctx), "unset")

	for value, want := range map[string]bool{
		"true":    false,
		"1":       false,
		"yes":     false,
		"on":      false,
		"false":   true,
		"0":       true,
		"no":      true,
		"off":     true,
		"":        true,
		"enabled": true,
	} {
		t.Run(value, func(t *testing.T) {
			assert.Equal(t, want, EnvOptedOut(env.Set(ctx, EnvVar, value)))
		})
	}
}
