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
			// accepted_version, the version whose question after a command
			// was answered yes, is ignored.
			name: "1.27 after a yes at the install prompt",
			data: `{"last_version": "1.27.0", "accepted_version": "1.27.0", "installing": true, "id": "machine"}`,
			want: State{LastVersion: "1.27.0", Installing: true, ID: "machine"},
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

			// A write keeps only the fields this CLI knows.
			_, err = SaveID(ctx, "other")
			require.NoError(t, err)
			data, err := os.ReadFile(filepath.Join(home, ".databricks", "aitools", stateFileName))
			require.NoError(t, err)
			assert.NotContains(t, string(data), "accepted_version")
			s, err = Load(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, s)
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

func TestMarkConsentAsked(t *testing.T) {
	for _, tc := range []struct {
		name       string
		data       string
		wantMarked bool
		want       State
	}{
		{name: "no state", wantMarked: true, want: State{ConsentAsked: true}},
		{name: "never answered", data: `{"id": "machine", "last_version": "1.30.0"}`, wantMarked: true, want: State{LastVersion: "1.30.0", ConsentAsked: true, ID: "machine"}},
		{name: "asked before", data: `{"id": "machine", "consent_asked": true}`, want: State{ConsentAsked: true, ID: "machine"}},
		{name: "consented", data: `{"id": "machine", "consented": true}`, want: State{Consented: true, ID: "machine"}},
		{name: "opted out", data: `{"id": "machine", "opted_out": true}`, want: State{OptedOut: true, ID: "machine"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			ctx := env.WithUserHomeDir(t.Context(), home)
			if tc.data != "" {
				writeStateFile(t, home, tc.data)
			}

			marked, err := MarkConsentAsked(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.wantMarked, marked)
			s, err := Load(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, s)
		})
	}
}

// A state file it cannot read may hold an opt-out, so it is kept.
func TestMarkConsentAskedKeepsACorruptFile(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	writeStateFile(t, home, "not json")

	marked, err := MarkConsentAsked(ctx)
	assert.ErrorContains(t, err, "parse ")
	assert.False(t, marked)
	data, err := os.ReadFile(filepath.Join(home, ".databricks", "aitools", stateFileName))
	require.NoError(t, err)
	assert.Equal(t, "not json", string(data))
}

// The record of the question survives every change of the state that keeps
// the rest of it, so it is never asked again.
func TestConsentAskedSurvivesTheOtherChanges(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	_, err := MarkConsentAsked(ctx)
	require.NoError(t, err)
	at := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	for name, change := range map[string]func() error{
		"SaveID":                 func() error { _, err := SaveID(ctx, "machine"); return err },
		"Record":                 func() error { return Record(ctx, "1.30.0", 1, "machine", at) },
		"StartAttempt":           func() error { _, err := StartAttempt(ctx, "1.30.0", 2, "machine", at); return err },
		"MarkInstalling":         func() error { return MarkInstalling(ctx) },
		"SaveInstallStateDigest": func() error { return SaveInstallStateDigest(ctx, "digest") },
		"GiveConsentIfUnknown":   func() error { _, err := GiveConsentIfUnknown(ctx); return err },
		"SetConsent":             func() error { _, err := SetConsent(ctx, ConsentOptedOut); return err },
		"Decline": func() error {
			t.Cleanup(ResetDeclined)
			_, err := Decline(ctx)
			return err
		},
	} {
		require.NoError(t, change(), name)
		s, err := Load(ctx)
		require.NoError(t, err)
		assert.True(t, s.ConsentAsked, "after %s", name)
	}
}

// A CLI from before the question at login reads a file that records it, and
// keeps reading an opt-out as one.
func TestOlderCLIReadsTheRecordOfTheQuestion(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	_, err := MarkConsentAsked(ctx)
	require.NoError(t, err)
	_, err = SetConsent(ctx, ConsentOptedOut)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(home, ".databricks", "aitools", stateFileName))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"consent_asked": true`)
	var s olderState
	require.NoError(t, json.Unmarshal(data, &s))
	assert.True(t, s.OptedOut)
}
