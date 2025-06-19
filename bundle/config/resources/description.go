package resources

type ResourceDescription struct {
	// Singular and plural name when used to refer to the configuration.
	SingularName string
	PluralName   string

	// Singular and plural title when used in summaries / terminal UI.
	SingularTitle string
	PluralTitle   string
}
