package events

// Corresponds to the `DatabricksCliBundleInitEvent` proto message in `databricks_cli_log.proto`
// as of 20 Dec 2024.
type BundleInitEvent struct {
	// UUID associated with the DAB itself. This is serialized into the DAB
	// when a user runs `databricks bundle init` and all subsequent deployments of
	// that DAB can then be associated with this init event.
	Uuid string `json:"uuid,omitempty"`

	// Name of the template initialized when the user ran `databricks bundle init`
	// This is only populated when the template is a first party template like
	// mlops-stacks or default-python.
	TemplateName BundleTemplate `json:"template_name,omitempty"`

	// Arguments used by the user to initialize the template. Only enum
	// values will be set here by the Databricks CLI.
	//
	// We use a generic map representation here because a bundle template's args are
	// managed in the template itself and maintaining a copy typed schema for it here
	// will be untenable in the long term.
	TemplateEnumArgs map[string]string `json:"template_enum_args,omitempty"`
}

type BundleTemplate string

const (
	BundleTemplateMlopsStacks   BundleTemplate = "mlops-stacks"
	BundleTemplateDefaultPython BundleTemplate = "default-python"
	BundleTemplateDefaultSql    BundleTemplate = "default-sql"
	BundleTemplateDbtSql        BundleTemplate = "dbt-sql"
	BundleTemplateCustom        BundleTemplate = "custom"
)
