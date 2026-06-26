// Package accessexplain traces why a principal can or cannot access a Unity
// Catalog securable and explains the verdict in plain language, with the exact
// GRANT to fix a denial.
//
// As with the auth doctor, the engine is pure: ParseSecurable builds the
// required-privilege chain, the cmd layer fills in the effective privileges and
// masking policies via the SDK, and Evaluate computes the verdict
// deterministically. Unity Catalog has no boolean "can P do A on S?" endpoint,
// so the verdict is computed by checking the effective-privilege set returned
// by Grants.GetEffective against the required set (USE CATALOG + USE SCHEMA +
// the leaf privilege).
package accessexplain

import (
	"fmt"
	"strings"
)

// Securable types, matching the SDK SecurableType enum string values.
const (
	SecurableCatalog = "CATALOG"
	SecurableSchema  = "SCHEMA"
	SecurableTable   = "TABLE"
)

// Privileges, matching the SDK Privilege enum string values.
const (
	PrivUseCatalog    = "USE_CATALOG"
	PrivUseSchema     = "USE_SCHEMA"
	PrivSelect        = "SELECT"
	PrivAllPrivileges = "ALL_PRIVILEGES"
)

// LevelSpec is one securable in the privilege chain the principal must satisfy,
// with the privilege required at that level. The cmd layer queries effective
// permissions per spec.
type LevelSpec struct {
	Type     string `json:"type"`
	FullName string `json:"full_name"`
	Needed   string `json:"needed"`
}

// HeldPrivilege is one effective privilege a principal holds at a securable,
// with its inheritance source (empty when granted directly).
type HeldPrivilege struct {
	Name              string `json:"name"`
	InheritedFromType string `json:"inherited_from_type,omitempty"`
	InheritedFromName string `json:"inherited_from_name,omitempty"`
}

// Level is a LevelSpec plus the privileges the principal effectively holds there.
type Level struct {
	LevelSpec
	Held []HeldPrivilege `json:"held"`
}

// Mask is a column-masking policy on the table. Targets lists the principals
// the policy applies to (empty means everyone). Applies is true when the policy
// definitely covers the requested principal (it targets everyone or names the
// principal directly); it is false when the policy targets other
// principals/groups and may only apply via group membership, which cannot be
// resolved locally. Such policies are reported (not dropped) with Applies=false
// so the explanation neither hides a possible mask nor asserts a false one.
type Mask struct {
	Column   string   `json:"column"`
	Policy   string   `json:"policy"`
	Function string   `json:"function,omitempty"`
	Targets  []string `json:"targets,omitempty"`
	Applies  bool     `json:"applies"`
}

// Input is the full trace the verdict is computed from.
type Input struct {
	Principal string  `json:"principal"`
	Securable string  `json:"securable"`
	Action    string  `json:"action"`
	Levels    []Level `json:"levels"`
	Masks     []Mask  `json:"masks,omitempty"`
}

// LevelResult is the per-level verdict.
type LevelResult struct {
	Type        string          `json:"type"`
	FullName    string          `json:"full_name"`
	Needed      string          `json:"needed"`
	Held        []HeldPrivilege `json:"held"`
	Satisfied   bool            `json:"satisfied"`
	SatisfiedBy string          `json:"satisfied_by,omitempty"`
}

// Verdict is the explained access decision.
type Verdict struct {
	Principal string `json:"principal"`
	// PrincipalKind is the resolved principal type ("user", "group", or
	// "service principal") when it could be verified via SCIM; empty otherwise.
	PrincipalKind string        `json:"principal_kind,omitempty"`
	Securable     string        `json:"securable"`
	Action        string        `json:"action"`
	Allowed       bool          `json:"allowed"`
	Levels        []LevelResult `json:"levels"`
	Masks         []Mask        `json:"masks,omitempty"`
	// Fixes are GRANT statements that resolve a denial, one per missing level.
	Fixes []string `json:"fixes,omitempty"`
}

// ParseSecurable builds the required-privilege chain for a dotted securable
// name. A 1-part name is a catalog, 2-part a schema, 3-part a table.
//
// The USE hierarchy is always required: USE CATALOG on the catalog and (for a
// schema or table) USE SCHEMA on the schema. action is an ADDITIONAL privilege
// required at the leaf securable. When action is empty it defaults to the
// natural read action: USE CATALOG for a catalog, USE SCHEMA for a schema,
// SELECT for a table. A non-default action adds a level rather than replacing
// the USE privilege, so e.g. CREATE_TABLE on a schema still requires USE SCHEMA.
func ParseSecurable(name, action string) ([]LevelSpec, error) {
	parts := strings.Split(name, ".")
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			return nil, fmt.Errorf("invalid securable %q: empty name component", name)
		}
	}

	action = normalizePrivilege(action)
	catalog := parts[0]

	switch len(parts) {
	case 1:
		return withLeafAction([]LevelSpec{
			{Type: SecurableCatalog, FullName: catalog, Needed: PrivUseCatalog},
		}, SecurableCatalog, catalog, action, PrivUseCatalog), nil
	case 2:
		schema := catalog + "." + parts[1]
		return withLeafAction([]LevelSpec{
			{Type: SecurableCatalog, FullName: catalog, Needed: PrivUseCatalog},
			{Type: SecurableSchema, FullName: schema, Needed: PrivUseSchema},
		}, SecurableSchema, schema, action, PrivUseSchema), nil
	case 3:
		schema := catalog + "." + parts[1]
		table := schema + "." + parts[2]
		// A table has no USE privilege of its own; the leaf action (SELECT by
		// default) is the table-level requirement on top of the USE hierarchy.
		return []LevelSpec{
			{Type: SecurableCatalog, FullName: catalog, Needed: PrivUseCatalog},
			{Type: SecurableSchema, FullName: schema, Needed: PrivUseSchema},
			{Type: SecurableTable, FullName: table, Needed: orDefault(action, PrivSelect)},
		}, nil
	default:
		return nil, fmt.Errorf("invalid securable %q: expected catalog[.schema[.table]], got %d parts", name, len(parts))
	}
}

// withLeafAction appends an extra leaf-level requirement when action is set and
// differs from the leaf's USE privilege (which the chain already requires).
func withLeafAction(specs []LevelSpec, leafType, leafName, action, usePriv string) []LevelSpec {
	if action != "" && action != usePriv {
		specs = append(specs, LevelSpec{Type: leafType, FullName: leafName, Needed: action})
	}
	return specs
}

// orDefault returns action when non-empty, else the default privilege.
func orDefault(action, def string) string {
	if action != "" {
		return action
	}
	return def
}

// normalizePrivilege canonicalizes a user-supplied privilege to the SDK enum
// form: upper-cased with spaces replaced by underscores, so "use schema",
// "USE SCHEMA", and "USE_SCHEMA" all become "USE_SCHEMA".
func normalizePrivilege(p string) string {
	return strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(p)), " ", "_")
}

// Evaluate computes the access verdict from the trace. A level is satisfied
// when the principal holds the needed privilege (directly or by inheritance) or
// ALL_PRIVILEGES. Access is allowed only when every level is satisfied.
func Evaluate(in Input) Verdict {
	v := Verdict{
		Principal: in.Principal,
		Securable: in.Securable,
		Action:    in.Action,
		Allowed:   true,
		Masks:     in.Masks,
	}

	for _, level := range in.Levels {
		held, by := satisfies(level)
		lr := LevelResult{
			Type:        level.Type,
			FullName:    level.FullName,
			Needed:      level.Needed,
			Held:        level.Held,
			Satisfied:   held,
			SatisfiedBy: by,
		}
		v.Levels = append(v.Levels, lr)
		if !held {
			v.Allowed = false
			v.Fixes = append(v.Fixes, grantStatement(level.Needed, level.Type, level.FullName, in.Principal))
		}
	}

	return v
}

// satisfies reports whether the level's needed privilege is held, and how.
func satisfies(level Level) (bool, string) {
	for _, h := range level.Held {
		if h.Name == level.Needed || h.Name == PrivAllPrivileges {
			return true, satisfiedByLabel(h)
		}
	}
	return false, ""
}

// satisfiedByLabel describes how a privilege was conveyed.
func satisfiedByLabel(h HeldPrivilege) string {
	if h.InheritedFromName != "" {
		return fmt.Sprintf("%s inherited from %s %s", h.Name, strings.ToLower(h.InheritedFromType), h.InheritedFromName)
	}
	return h.Name + " granted directly"
}

// grantStatement builds the UC GRANT that confers the needed privilege. The
// principal is always backtick-quoted (it is an identifier in GRANT TO); the
// securable name's parts are quoted only when they need it. Embedded backticks
// are escaped by doubling, so the statement stays valid SQL for unusual names.
func grantStatement(privilege, securableType, fullName, principal string) string {
	return fmt.Sprintf("GRANT %s ON %s %s TO %s", grantPrivilege(privilege), securableType, quoteSecurable(fullName), quoteIdentifier(principal))
}

// grantPrivilege renders an SDK privilege enum (USE_SCHEMA) as GRANT syntax
// (USE SCHEMA).
func grantPrivilege(privilege string) string {
	return strings.ReplaceAll(privilege, "_", " ")
}

// quoteSecurable backtick-quotes each dot-separated part of a securable name
// that needs quoting, leaving simple identifiers unquoted for readability.
func quoteSecurable(fullName string) string {
	parts := strings.Split(fullName, ".")
	for i, p := range parts {
		if needsQuoting(p) {
			parts[i] = quoteIdentifier(p)
		}
	}
	return strings.Join(parts, ".")
}

// quoteIdentifier backtick-quotes an identifier, escaping embedded backticks by
// doubling them per the SQL identifier-quoting rule.
func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// needsQuoting reports whether a securable name part must be delimited: it is
// empty, contains a character outside [A-Za-z0-9_], or is all digits (Databricks
// SQL requires all-digit identifiers to be quoted).
func needsQuoting(part string) bool {
	if part == "" {
		return true
	}
	allDigits := true
	for _, r := range part {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if !isLetter && !isDigit && r != '_' {
			return true
		}
		if !isDigit {
			allDigits = false
		}
	}
	// Databricks SQL requires an all-digit identifier to be delimited.
	return allDigits
}
