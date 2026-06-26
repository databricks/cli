package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/databricks/cli/experimental/accessexplain"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeReader is an accessReader stub keyed by securable full name.
type fakeReader struct {
	user        string
	effective   map[string][]accessexplain.HeldPrivilege
	masks       []accessexplain.Mask
	legacyMasks []accessexplain.Mask
	// principal resolution. Defaults to a found user; set principalMissing to
	// simulate a non-existent principal, or resolveErr for an unverifiable lookup.
	principalKind    string
	principalMissing bool
	resolveErr       error
	maskErr          error
}

func (f *fakeReader) ResolvePrincipal(ctx context.Context, name string) (string, bool, error) {
	if f.resolveErr != nil {
		return "", false, f.resolveErr
	}
	if f.principalMissing {
		return "", false, nil
	}
	kind := f.principalKind
	if kind == "" {
		kind = "user"
	}
	return kind, true, nil
}

func (f *fakeReader) Effective(ctx context.Context, securableType, fullName, principal string) ([]accessexplain.HeldPrivilege, error) {
	return f.effective[fullName], nil
}

func (f *fakeReader) ColumnMasks(ctx context.Context, tableFullName, principal string) ([]accessexplain.Mask, error) {
	return f.masks, f.maskErr
}

func (f *fakeReader) LegacyColumnMasks(ctx context.Context, tableFullName string) ([]accessexplain.Mask, error) {
	return f.legacyMasks, nil
}

func (f *fakeReader) CurrentUser(ctx context.Context) (string, error) {
	return f.user, nil
}

func held(names ...string) []accessexplain.HeldPrivilege {
	var out []accessexplain.HeldPrivilege
	for _, n := range names {
		out = append(out, accessexplain.HeldPrivilege{Name: n})
	}
	return out
}

func TestExplainAllowed(t *testing.T) {
	r := &fakeReader{
		effective: map[string][]accessexplain.HeldPrivilege{
			"prod":                    held(accessexplain.PrivUseCatalog),
			"prod.sales":              held(accessexplain.PrivUseSchema),
			"prod.sales.transactions": held(accessexplain.PrivSelect),
		},
	}
	v, err := explain(t.Context(), r, "prod.sales.transactions", "alice@databricks.test", "")
	require.NoError(t, err)
	assert.True(t, v.Allowed)
	assert.Equal(t, "alice@databricks.test", v.Principal)
	assert.Equal(t, accessexplain.PrivSelect, v.Action)
}

func TestExplainDeniedMissingUseSchema(t *testing.T) {
	r := &fakeReader{
		effective: map[string][]accessexplain.HeldPrivilege{
			"prod":                    held(accessexplain.PrivUseCatalog),
			"prod.sales.transactions": held(accessexplain.PrivSelect),
		},
	}
	v, err := explain(t.Context(), r, "prod.sales.transactions", "alice@databricks.test", "")
	require.NoError(t, err)
	assert.False(t, v.Allowed)
	require.Len(t, v.Fixes, 1)
	assert.Equal(t, "GRANT USE SCHEMA ON SCHEMA prod.sales TO `alice@databricks.test`", v.Fixes[0])
}

func TestExplainDefaultsToCurrentUser(t *testing.T) {
	r := &fakeReader{
		user: "me@databricks.test",
		effective: map[string][]accessexplain.HeldPrivilege{
			"prod":       held(accessexplain.PrivUseCatalog),
			"prod.sales": held(accessexplain.PrivUseSchema),
		},
	}
	v, err := explain(t.Context(), r, "prod.sales", "", "")
	require.NoError(t, err)
	assert.Equal(t, "me@databricks.test", v.Principal)
	assert.True(t, v.Allowed)
}

func TestExplainSurfacesMasks(t *testing.T) {
	r := &fakeReader{
		effective: map[string][]accessexplain.HeldPrivilege{
			"prod":                    held(accessexplain.PrivUseCatalog),
			"prod.sales":              held(accessexplain.PrivUseSchema),
			"prod.sales.transactions": held(accessexplain.PrivSelect),
		},
		masks: []accessexplain.Mask{{Column: "ssn", Policy: "pii_mask"}},
	}
	v, err := explain(t.Context(), r, "prod.sales.transactions", "alice@databricks.test", "")
	require.NoError(t, err)
	assert.True(t, v.Allowed)
	require.Len(t, v.Masks, 1)
	assert.Equal(t, "pii_mask", v.Masks[0].Policy)
}

func TestExplainMergesLegacyAndPolicyMasks(t *testing.T) {
	r := &fakeReader{
		effective: map[string][]accessexplain.HeldPrivilege{
			"prod":                    held(accessexplain.PrivUseCatalog),
			"prod.sales":              held(accessexplain.PrivUseSchema),
			"prod.sales.transactions": held(accessexplain.PrivSelect),
		},
		masks:       []accessexplain.Mask{{Column: "ssn", Policy: "pii_mask"}},
		legacyMasks: []accessexplain.Mask{{Column: "email", Function: "main.default.mask_email"}, {Column: "ssn", Function: "legacy_ssn"}},
	}
	v, err := explain(t.Context(), r, "prod.sales.transactions", "alice@databricks.test", "")
	require.NoError(t, err)
	// ssn appears once (policy wins over legacy), email from the legacy mask.
	require.Len(t, v.Masks, 2)
	byCol := map[string]accessexplain.Mask{}
	for _, m := range v.Masks {
		byCol[m.Column] = m
	}
	assert.Equal(t, "pii_mask", byCol["ssn"].Policy)
	assert.Equal(t, "main.default.mask_email", byCol["email"].Function)
}

func TestColumnMaskFromPolicy(t *testing.T) {
	colMask := func(name, col string, to, except []string) catalog.PolicyInfo {
		return catalog.PolicyInfo{
			Name:             name,
			PolicyType:       catalog.PolicyTypePolicyTypeColumnMask,
			ColumnMask:       &catalog.ColumnMaskOptions{OnColumn: col, FunctionName: "f"},
			ToPrincipals:     to,
			ExceptPrincipals: except,
		}
	}

	t.Run("targets everyone", func(t *testing.T) {
		m, ok := columnMaskFromPolicy(colMask("p", "ssn", nil, nil), "alice@x")
		require.True(t, ok)
		assert.Equal(t, "ssn", m.Column)
		assert.Empty(t, m.Targets)
	})

	t.Run("group-targeted is kept with targets", func(t *testing.T) {
		m, ok := columnMaskFromPolicy(colMask("p", "ssn", []string{"data-scientists"}, nil), "alice@x")
		require.True(t, ok)
		assert.Equal(t, []string{"data-scientists"}, m.Targets)
	})

	t.Run("excepted principal is dropped", func(t *testing.T) {
		_, ok := columnMaskFromPolicy(colMask("p", "ssn", nil, []string{"alice@x"}), "alice@x")
		assert.False(t, ok)
	})

	t.Run("row filter policy is not a column mask", func(t *testing.T) {
		_, ok := columnMaskFromPolicy(catalog.PolicyInfo{PolicyType: catalog.PolicyTypePolicyTypeRowFilter}, "alice@x")
		assert.False(t, ok)
	})
}

func TestExplainInvalidSecurable(t *testing.T) {
	r := &fakeReader{}
	_, err := explain(t.Context(), r, "a..b", "alice@databricks.test", "")
	assert.Error(t, err)
}

func TestExplainMaskErrorIsNonFatal(t *testing.T) {
	// A failure listing masks (e.g. no READ METADATA) must not abort the
	// verdict, which is the command's core value.
	r := &fakeReader{
		maskErr: errors.New("User does not have READ METADATA on Table"),
		effective: map[string][]accessexplain.HeldPrivilege{
			"prod":                    held(accessexplain.PrivUseCatalog),
			"prod.sales":              held(accessexplain.PrivUseSchema),
			"prod.sales.transactions": held(accessexplain.PrivSelect),
		},
	}
	v, err := explain(t.Context(), r, "prod.sales.transactions", "alice@databricks.test", "")
	require.NoError(t, err)
	assert.True(t, v.Allowed)
	assert.Empty(t, v.Masks)
}

func TestExplainPrincipalNotFound(t *testing.T) {
	r := &fakeReader{principalMissing: true}
	_, err := explain(t.Context(), r, "prod.sales.transactions", "typo@databricks.test", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExplainPrincipalUnverifiableProceeds(t *testing.T) {
	// A list-permission error must not block the explanation.
	r := &fakeReader{
		resolveErr: errors.New("permission denied"),
		effective: map[string][]accessexplain.HeldPrivilege{
			"prod":       held(accessexplain.PrivUseCatalog),
			"prod.sales": held(accessexplain.PrivUseSchema),
		},
	}
	v, err := explain(t.Context(), r, "prod.sales", "alice@databricks.test", "")
	require.NoError(t, err)
	assert.True(t, v.Allowed)
	assert.Empty(t, v.PrincipalKind) // unverified, so no kind reported
}

func TestExplainReportsPrincipalKind(t *testing.T) {
	r := &fakeReader{
		principalKind: "group",
		effective: map[string][]accessexplain.HeldPrivilege{
			"prod":       held(accessexplain.PrivUseCatalog),
			"prod.sales": held(accessexplain.PrivUseSchema),
		},
	}
	v, err := explain(t.Context(), r, "prod.sales", "data-scientists", "")
	require.NoError(t, err)
	assert.Equal(t, "group", v.PrincipalKind)
}

func TestExplainCurrentUserNotResolved(t *testing.T) {
	// The current-user path does not run principal resolution.
	r := &fakeReader{
		user:             "me@databricks.test",
		principalMissing: true,
		effective: map[string][]accessexplain.HeldPrivilege{
			"prod":       held(accessexplain.PrivUseCatalog),
			"prod.sales": held(accessexplain.PrivUseSchema),
		},
	}
	v, err := explain(t.Context(), r, "prod.sales", "", "")
	require.NoError(t, err)
	assert.Equal(t, "me@databricks.test", v.Principal)
}

func newRenderCmd(t *testing.T, output flags.Output) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{}
	out := output
	cmd.Flags().Var(&out, "output", "")
	cmd.SetContext(t.Context())
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	return cmd, buf
}

var deniedVerdict = accessexplain.Verdict{
	Principal: "alice@databricks.test",
	Securable: "prod.sales.transactions",
	Action:    "SELECT",
	Allowed:   false,
	Levels: []accessexplain.LevelResult{
		{Type: "CATALOG", FullName: "prod", Needed: "USE_CATALOG", Satisfied: true, SatisfiedBy: "USE_CATALOG granted on this catalog"},
		{Type: "SCHEMA", FullName: "prod.sales", Needed: "USE_SCHEMA", Satisfied: false},
		{Type: "TABLE", FullName: "prod.sales.transactions", Needed: "SELECT", Satisfied: true, SatisfiedBy: "SELECT granted on this table"},
	},
	Masks: []accessexplain.Mask{{Column: "ssn", Policy: "pii_mask", Applies: true}},
	Fixes: []string{"GRANT USE SCHEMA ON SCHEMA prod.sales TO `alice@databricks.test`"},
}

func TestRenderVerdictText(t *testing.T) {
	cmd, buf := newRenderCmd(t, flags.OutputText)
	require.NoError(t, renderVerdict(cmd.Context(), cmd, deniedVerdict))

	s := buf.String()
	assert.Contains(t, s, "DENIED")
	assert.Contains(t, s, "missing USE_SCHEMA")
	assert.Contains(t, s, "column ssn is masked by policy pii_mask")
	assert.Contains(t, s, "GRANT USE SCHEMA ON SCHEMA prod.sales TO `alice@databricks.test`")
}

func TestRenderVerdictGroupTargetedMask(t *testing.T) {
	cmd, buf := newRenderCmd(t, flags.OutputText)
	v := accessexplain.Verdict{
		Principal: "alice@databricks.test",
		Securable: "prod.sales.transactions",
		Action:    "SELECT",
		Allowed:   true,
		Masks:     []accessexplain.Mask{{Column: "ssn", Policy: "pii_mask", Targets: []string{"data-scientists"}, Applies: false}},
	}
	require.NoError(t, renderVerdict(cmd.Context(), cmd, v))
	s := buf.String()
	// A group-targeted policy is reported as a possibility, not asserted.
	assert.Contains(t, s, "column ssn may be masked by policy pii_mask (targets data-scientists)")
}

func TestRenderVerdictJSON(t *testing.T) {
	cmd, buf := newRenderCmd(t, flags.OutputJSON)
	require.NoError(t, renderVerdict(cmd.Context(), cmd, deniedVerdict))

	var got accessexplain.Verdict
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.False(t, got.Allowed)
	assert.Len(t, got.Levels, 3)
	assert.Len(t, got.Fixes, 1)
}
