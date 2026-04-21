package resources

// Securable identifies a UC object that a grant applies to.
type Securable struct {
	// Type of the securable (catalog, schema, table, volume, external_location, ...).
	Type string `json:"type"`

	// Name of the securable.
	Name string `json:"name"`
}

// Grant assigns privileges on a securable to a principal.
type Grant struct {
	// Securable is the object receiving the grant.
	Securable Securable `json:"securable"`

	// Principal is the UC/account-level principal (user, group, SP) name.
	Principal string `json:"principal"`

	// Privileges is the list of UC privileges being granted
	// (e.g., USE_CATALOG, USE_SCHEMA, SELECT, MODIFY).
	Privileges []string `json:"privileges"`
}
