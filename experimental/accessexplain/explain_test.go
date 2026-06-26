package accessexplain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSecurable(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		action string
		want   []LevelSpec
	}{
		{
			name: "catalog",
			in:   "prod",
			want: []LevelSpec{{SecurableCatalog, "prod", PrivUseCatalog}},
		},
		{
			name: "schema",
			in:   "prod.sales",
			want: []LevelSpec{
				{SecurableCatalog, "prod", PrivUseCatalog},
				{SecurableSchema, "prod.sales", PrivUseSchema},
			},
		},
		{
			name: "table",
			in:   "prod.sales.transactions",
			want: []LevelSpec{
				{SecurableCatalog, "prod", PrivUseCatalog},
				{SecurableSchema, "prod.sales", PrivUseSchema},
				{SecurableTable, "prod.sales.transactions", PrivSelect},
			},
		},
		{
			name:   "table with action override",
			in:     "prod.sales.transactions",
			action: "modify",
			want: []LevelSpec{
				{SecurableCatalog, "prod", PrivUseCatalog},
				{SecurableSchema, "prod.sales", PrivUseSchema},
				{SecurableTable, "prod.sales.transactions", "MODIFY"},
			},
		},
		{
			// A non-default action on a schema must NOT drop USE SCHEMA.
			name:   "schema with action keeps USE SCHEMA",
			in:     "prod.sales",
			action: "create table",
			want: []LevelSpec{
				{SecurableCatalog, "prod", PrivUseCatalog},
				{SecurableSchema, "prod.sales", PrivUseSchema},
				{SecurableSchema, "prod.sales", "CREATE_TABLE"},
			},
		},
		{
			name:   "catalog with action keeps USE CATALOG",
			in:     "prod",
			action: "CREATE_SCHEMA",
			want: []LevelSpec{
				{SecurableCatalog, "prod", PrivUseCatalog},
				{SecurableCatalog, "prod", "CREATE_SCHEMA"},
			},
		},
		{
			name:   "action equal to use privilege is not duplicated",
			in:     "prod.sales",
			action: "use schema",
			want: []LevelSpec{
				{SecurableCatalog, "prod", PrivUseCatalog},
				{SecurableSchema, "prod.sales", PrivUseSchema},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSecurable(tt.in, tt.action)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSecurableInvalid(t *testing.T) {
	for _, in := range []string{"", "a.", ".b", "a..b", "a.b.c.d"} {
		_, err := ParseSecurable(in, "")
		assert.Error(t, err, "expected error for %q", in)
	}
}

// tableLevels builds a 3-level table trace with the given held privileges.
func tableLevels(catalogP, schemaP, tableP []HeldPrivilege) []Level {
	return []Level{
		{LevelSpec: LevelSpec{SecurableCatalog, "prod", PrivUseCatalog}, Held: catalogP},
		{LevelSpec: LevelSpec{SecurableSchema, "prod.sales", PrivUseSchema}, Held: schemaP},
		{LevelSpec: LevelSpec{SecurableTable, "prod.sales.transactions", PrivSelect}, Held: tableP},
	}
}

func held(names ...string) []HeldPrivilege {
	var out []HeldPrivilege
	for _, n := range names {
		out = append(out, HeldPrivilege{Name: n})
	}
	return out
}

func TestEvaluateAllowed(t *testing.T) {
	in := Input{
		Principal: "alice@databricks.test",
		Securable: "prod.sales.transactions",
		Action:    PrivSelect,
		Levels: tableLevels(
			held(PrivUseCatalog),
			held(PrivUseSchema),
			held(PrivSelect),
		),
	}
	v := Evaluate(in)
	assert.True(t, v.Allowed)
	assert.Empty(t, v.Fixes)
	for _, l := range v.Levels {
		assert.True(t, l.Satisfied)
	}
}

func TestEvaluateMissingUseSchema(t *testing.T) {
	// The classic gotcha: SELECT on the table but no USE SCHEMA on the schema.
	in := Input{
		Principal: "alice@databricks.test",
		Securable: "prod.sales.transactions",
		Action:    PrivSelect,
		Levels: tableLevels(
			held(PrivUseCatalog),
			nil,
			held(PrivSelect),
		),
	}
	v := Evaluate(in)
	assert.False(t, v.Allowed)
	require.Len(t, v.Fixes, 1)
	assert.Equal(t, "GRANT USE SCHEMA ON SCHEMA prod.sales TO `alice@databricks.test`", v.Fixes[0])

	// The schema level is the only unsatisfied one.
	assert.True(t, v.Levels[0].Satisfied)
	assert.False(t, v.Levels[1].Satisfied)
	assert.True(t, v.Levels[2].Satisfied)
}

func TestEvaluateInheritedSatisfaction(t *testing.T) {
	in := Input{
		Principal: "alice@databricks.test",
		Securable: "prod.sales.transactions",
		Action:    PrivSelect,
		Levels: tableLevels(
			held(PrivUseCatalog),
			held(PrivUseSchema),
			[]HeldPrivilege{{Name: PrivSelect, InheritedFromType: "CATALOG", InheritedFromName: "prod"}},
		),
	}
	v := Evaluate(in)
	assert.True(t, v.Allowed)
	assert.Equal(t, "SELECT inherited from catalog prod", v.Levels[2].SatisfiedBy)
}

func TestEvaluateAllPrivileges(t *testing.T) {
	in := Input{
		Principal: "alice@databricks.test",
		Securable: "prod.sales.transactions",
		Action:    PrivSelect,
		Levels: tableLevels(
			held(PrivAllPrivileges),
			held(PrivAllPrivileges),
			held(PrivAllPrivileges),
		),
	}
	v := Evaluate(in)
	assert.True(t, v.Allowed)
	assert.Contains(t, v.Levels[2].SatisfiedBy, "ALL_PRIVILEGES")
}

func TestEvaluateCarriesMasks(t *testing.T) {
	in := Input{
		Principal: "alice@databricks.test",
		Securable: "prod.sales.transactions",
		Action:    PrivSelect,
		Levels:    tableLevels(held(PrivUseCatalog), held(PrivUseSchema), held(PrivSelect)),
		Masks:     []Mask{{Column: "ssn", Policy: "pii_mask", Function: "main.default.mask_ssn"}},
	}
	v := Evaluate(in)
	assert.True(t, v.Allowed)
	require.Len(t, v.Masks, 1)
	assert.Equal(t, "ssn", v.Masks[0].Column)
}

func TestGrantStatement(t *testing.T) {
	assert.Equal(t, "GRANT USE CATALOG ON CATALOG prod TO `g`", grantStatement(PrivUseCatalog, SecurableCatalog, "prod", "g"))
	assert.Equal(t, "GRANT SELECT ON TABLE prod.s.t TO `u@x`", grantStatement(PrivSelect, SecurableTable, "prod.s.t", "u@x"))
	// A backtick in the principal is escaped by doubling.
	assert.Equal(t, "GRANT SELECT ON TABLE prod.s.t TO `we``ird`", grantStatement(PrivSelect, SecurableTable, "prod.s.t", "we`ird"))
	// A securable name part with special characters is quoted; simple parts are not.
	assert.Equal(t, "GRANT SELECT ON TABLE prod.`my-schema`.t TO `g`", grantStatement(PrivSelect, SecurableTable, "prod.my-schema.t", "g"))
	// An all-digit name part must be delimited.
	assert.Equal(t, "GRANT SELECT ON TABLE prod.`123`.t TO `g`", grantStatement(PrivSelect, SecurableTable, "prod.123.t", "g"))
}
