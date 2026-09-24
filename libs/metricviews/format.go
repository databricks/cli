package metricviews

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// DecimalPlacesType is "max", "exact", or "all" (YAML lowercase).
type DecimalPlacesType string

// DecimalPlaces configures decimal rendering for numeric formats.
type DecimalPlaces struct {
	Type   DecimalPlacesType `yaml:"type" json:"type"`
	Places *int              `yaml:"places,omitempty" json:"places,omitempty"`
}

func (d *DecimalPlaces) UnmarshalYAML(node *yaml.Node) error {
	if err := requireYAMLFields(node, "type"); err != nil {
		return err
	}
	type alias DecimalPlaces
	if err := node.Decode((*alias)(d)); err != nil {
		return err
	}
	switch d.Type {
	case "max", "exact", "all":
		return nil
	default:
		return fmt.Errorf("unsupported decimal_places.type %q", d.Type)
	}
}

// ColumnFormat is the internally-tagged column format union. Only the fields
// valid for the active Type are populated; the rest stay empty and are omitted
// on marshal. We model the YAML shape only (not the backend's JSON metadata form).
type ColumnFormat struct {
	Type string `yaml:"type" json:"type"`

	// number, currency, percentage, byte
	DecimalPlaces      *DecimalPlaces `yaml:"decimal_places,omitempty" json:"decimal_places,omitempty"`
	HideGroupSeparator *bool          `yaml:"hide_group_separator,omitempty" json:"hide_group_separator,omitempty"`

	// number, currency
	Abbreviation *string `yaml:"abbreviation,omitempty" json:"abbreviation,omitempty"`

	// currency
	CurrencyCode *string `yaml:"currency_code,omitempty" json:"currency_code,omitempty"`

	// date, date_time
	DateFormat *string `yaml:"date_format,omitempty" json:"date_format,omitempty"`
	// date_time
	TimeFormat *string `yaml:"time_format,omitempty" json:"time_format,omitempty"`
	// date, date_time
	LeadingZeros *bool `yaml:"leading_zeros,omitempty" json:"leading_zeros,omitempty"`
}

// UnmarshalYAML checks the format's tag and fields before decoding it.
func (f *ColumnFormat) UnmarshalYAML(node *yaml.Node) error {
	if err := requireYAMLFields(node, "type"); err != nil {
		return err
	}

	var typ string
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "type" {
			if err := node.Content[i+1].Decode(&typ); err != nil {
				return fmt.Errorf("decoding format type: %w", err)
			}
			break
		}
	}

	allowed := map[string]bool{"type": true}
	var required []string
	switch typ {
	case "number":
		allowed["decimal_places"], allowed["hide_group_separator"], allowed["abbreviation"] = true, true, true
	case "currency":
		allowed["decimal_places"], allowed["hide_group_separator"], allowed["abbreviation"], allowed["currency_code"] = true, true, true, true
		required = []string{"currency_code"}
	case "percentage", "byte":
		allowed["decimal_places"], allowed["hide_group_separator"] = true, true
	case "date":
		allowed["date_format"], allowed["leading_zeros"] = true, true
		required = []string{"date_format"}
	case "date_time":
		allowed["date_format"], allowed["time_format"], allowed["leading_zeros"] = true, true, true
		required = []string{"date_format", "time_format"}
	default:
		return fmt.Errorf("unsupported format type %q", typ)
	}
	if err := requireYAMLFields(node, required...); err != nil {
		return err
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		field := node.Content[i].Value
		if !allowed[field] {
			return fmt.Errorf("field %q is not valid for format type %q", field, typ)
		}
	}

	type alias ColumnFormat
	return node.Decode((*alias)(f))
}
