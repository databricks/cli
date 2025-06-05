package resources

import "strings"

type ResourceDescription struct {
	// SingularName is the canonical identifier of the resource type.
	// It is the only piece of static data stored; all other views are derived.
	SingularName string
}

// PluralName returns the plural snake-case identifier of the resource type.
// Currently all resource names pluralise by suffixing an "s".
func (r ResourceDescription) PluralName() string {
	return pluralize(r.SingularName)
}

// SingularTitle returns a human-readable singular title (e.g. "Model Serving Endpoint").
func (r ResourceDescription) SingularTitle() string {
	return humanize(r.SingularName)
}

// PluralTitle returns a human-readable plural title (e.g. "Model Serving Endpoints").
func (r ResourceDescription) PluralTitle() string {
	return humanize(r.PluralName())
}

// TerraformResourceName returns the Terraform resource name corresponding to this resource.
func (r ResourceDescription) TerraformResourceName() string {
	switch r.SingularName {
	case "experiment", "model":
		return "databricks_mlflow_" + r.SingularName
	default:
		return "databricks_" + r.SingularName
	}
}

// pluralize performs the naïve pluralisation by appending "s" unless already present.
func pluralize(s string) string {
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}

// humanize converts snake_case to Title Case with spaces.
func humanize(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
