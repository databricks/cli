package protos

type BundleInitEvent struct {
	// UUID associated with the DAB itself. This is serialized into the DAB
	// when a user runs `databricks bundle init` and all subsequent deployments of
	// that DAB can then be associated with this init event.
	BundleUuid string `json:"bundle_uuid,omitempty"`

	// Name of the template initialized when the user ran `databricks bundle init`
	// This is only populated when the template is a first party template like
	// mlops-stacks or default-python.
	TemplateName string `json:"template_name,omitempty"`

	// Arguments used by the user to initialize the template. Only enum
	// values will be set here by the Databricks CLI.
	//
	// We use a generic map representation here because a bundle template's args are
	// managed in the template itself and maintaining a copy typed schema for it here
	// will be untenable in the long term.
	TemplateEnumArgs []BundleInitTemplateEnumArg `json:"template_enum_args,omitempty"`
}

type BundleInitTemplateEnumArg struct {
	// Valid key values for the template. These correspond to the keys specified in
	// the "properties" section of the `databricks_template_schema.json` file.
	//
	// Note: `databricks_template_schema.json` contains a JSON schema type specification
	// for the arguments that the template accepts.
	Key string `json:"key"`

	// Value that the user set for the field. This is only populated for properties
	// that have the "enum" field specified in the JSON schema type specification.
	//
	// The Databricks CLI ensures that the value here is one of the "enum" values from
	// the template specification.
	Value string `json:"value"`
}
