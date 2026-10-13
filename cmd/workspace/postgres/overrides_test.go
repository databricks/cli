package postgres

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/duration"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cmdWithJSON(t *testing.T, raw string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	var jf flags.JsonFlag
	cmd.Flags().Var(&jf, "json", "JSON body")
	if raw != "" {
		require.NoError(t, jf.Set(raw))
	}
	return cmd
}

func TestRejectWrappedRoleJSON(t *testing.T) {
	t.Run("rejects wrapped {role: ...}", func(t *testing.T) {
		cmd := cmdWithJSON(t, `{"role":{"spec":{"identity_type":"SERVICE_PRINCIPAL"}}}`)
		err := rejectWrappedRoleJSON(cmd)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "should NOT be wrapped")
		assert.Contains(t, err.Error(), `databricks postgres create-role`)
	})

	t.Run("passes when body has spec at top level", func(t *testing.T) {
		cmd := cmdWithJSON(t, `{"spec":{"identity_type":"SERVICE_PRINCIPAL"}}`)
		assert.NoError(t, rejectWrappedRoleJSON(cmd))
	})

	t.Run("passes when --json was not provided", func(t *testing.T) {
		cmd := cmdWithJSON(t, "")
		assert.NoError(t, rejectWrappedRoleJSON(cmd))
	})

	t.Run("passes through non-object JSON to the generated diagnostics path", func(t *testing.T) {
		cmd := cmdWithJSON(t, `"not-an-object"`)
		assert.NoError(t, rejectWrappedRoleJSON(cmd))
	})

	t.Run("fails loudly when --json flag is absent on the command", func(t *testing.T) {
		// Internal invariant: postgres create-role is a generated command and
		// always has a --json flag. If a future codegen change drops it, this
		// override is wired to the wrong command and should fail loudly so the
		// regression is caught rather than silently disabling the guard.
		err := rejectWrappedRoleJSON(&cobra.Command{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal:")
	})
}

const sevenDays = 7 * 24 * time.Hour

func TestReconcileBranchExpiration(t *testing.T) {
	tests := []struct {
		name            string
		spec            *postgres.BranchSpec
		ttlChanged      bool
		ttl             string
		noExpiryChanged bool
		noExpiry        bool
		jsonExpiration  bool
		wantErr         string
		check           func(t *testing.T, spec *postgres.BranchSpec)
	}{
		{
			name:    "no flags, nil spec -> expiration required",
			wantErr: "a branch expiration is required",
		},
		{
			name:       "ttl API form",
			ttlChanged: true,
			ttl:        "604800s",
			check: func(t *testing.T, spec *postgres.BranchSpec) {
				require.NotNil(t, spec.Ttl)
				assert.Equal(t, sevenDays, spec.Ttl.AsDuration())
				assert.False(t, spec.NoExpiry)
			},
		},
		{
			name:       "ttl Go duration form",
			ttlChanged: true,
			ttl:        "168h",
			check: func(t *testing.T, spec *postgres.BranchSpec) {
				require.NotNil(t, spec.Ttl)
				assert.Equal(t, sevenDays, spec.Ttl.AsDuration())
			},
		},
		{
			name:            "no-expiry flag",
			noExpiryChanged: true,
			noExpiry:        true,
			check: func(t *testing.T, spec *postgres.BranchSpec) {
				assert.True(t, spec.NoExpiry)
			},
		},
		{
			name:    "non-expiration spec field is not an expiration source",
			spec:    &postgres.BranchSpec{SourceBranch: "projects/p/branches/main"},
			wantErr: "a branch expiration is required",
		},
		{
			name:           "json expiration alone leaves spec untouched",
			jsonExpiration: true,
			check: func(t *testing.T, spec *postgres.BranchSpec) {
				// spec stays nil so the generated RunE merges --json's expiration.
				assert.Nil(t, spec)
			},
		},
		{
			name:            "ttl and no-expiry conflict",
			ttlChanged:      true,
			ttl:             "604800s",
			noExpiryChanged: true,
			noExpiry:        true,
			wantErr:         "set more than once",
		},
		{
			name:           "ttl and json expiration conflict",
			ttlChanged:     true,
			ttl:            "604800s",
			jsonExpiration: true,
			wantErr:        "set more than once",
		},
		{
			name:            "no-expiry and json expiration conflict",
			noExpiryChanged: true,
			noExpiry:        true,
			jsonExpiration:  true,
			wantErr:         "set more than once",
		},
		{
			name:            "no-expiry=false is rejected",
			noExpiryChanged: true,
			noExpiry:        false,
			wantErr:         "--no-expiry=false is not valid",
		},
		{
			name:       "invalid ttl",
			ttlChanged: true,
			ttl:        "7x",
			wantErr:    "invalid --ttl",
		},
		{
			name:       "empty ttl",
			ttlChanged: true,
			ttl:        "",
			wantErr:    "must not be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := reconcileBranchExpiration(tc.spec, tc.ttlChanged, tc.ttl, tc.noExpiryChanged, tc.noExpiry, tc.jsonExpiration)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			tc.check(t, spec)
		})
	}
}

func TestParseTTL(t *testing.T) {
	valid := map[string]time.Duration{
		"604800s": sevenDays,
		"168h":    sevenDays,
		"7d":      sevenDays,
		"1w":      sevenDays,
		"1.5s":    1500 * time.Millisecond,
		"12d":     12 * 24 * time.Hour,
		"3w":      3 * 7 * 24 * time.Hour,
		"1.5d":    36 * time.Hour,
		"1w3d":    10 * 24 * time.Hour,
		"7d12h":   7*24*time.Hour + 12*time.Hour,
	}
	for in, want := range valid {
		t.Run("valid/"+in, func(t *testing.T) {
			d, err := parseTTL(in)
			require.NoError(t, err)
			require.NotNil(t, d)
			assert.Equal(t, want, d.AsDuration())
		})
	}

	for _, in := range []string{"", "abc", "0s", "0d", "-1h", "7x", "7dd", "d", "100000000000000000d", "9999999999w"} {
		t.Run("invalid/"+in, func(t *testing.T) {
			_, err := parseTTL(in)
			assert.Error(t, err)
		})
	}
}

func TestSpecHasExpiration(t *testing.T) {
	tests := []struct {
		name string
		spec *postgres.BranchSpec
		want bool
	}{
		{"nil spec", nil, false},
		{"empty spec", &postgres.BranchSpec{}, false},
		{"ttl set", &postgres.BranchSpec{Ttl: duration.New(sevenDays)}, true},
		{"expire_time set", &postgres.BranchSpec{ExpireTime: sdktime.New(time.Now())}, true},
		{"no_expiry true", &postgres.BranchSpec{NoExpiry: true}, true},
		{"no_expiry false, unset", &postgres.BranchSpec{NoExpiry: false}, false},
		{"no_expiry false, explicitly set", &postgres.BranchSpec{NoExpiry: false, ForceSendFields: []string{"NoExpiry"}}, true},
		{"non-expiration field only", &postgres.BranchSpec{SourceBranch: "projects/p/branches/main"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, specHasExpiration(tc.spec))
		})
	}
}

func TestCreateBranchOverrideWiring(t *testing.T) {
	cmd := newCreateBranch()
	assert.NotNil(t, cmd.Flags().Lookup("ttl"))
	assert.NotNil(t, cmd.Flags().Lookup("no-expiry"))
	assert.Contains(t, cmd.Long, "--ttl")
	assert.Contains(t, cmd.Long, "604800s")
}
