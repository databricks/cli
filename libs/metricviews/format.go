package metricviews

// DecimalPlacesType is "max", "exact", or "all" (YAML lowercase).
type DecimalPlacesType string

// DecimalPlaces configures decimal rendering for numeric formats.
type DecimalPlaces struct {
	Type   DecimalPlacesType `yaml:"type" json:"type"`
	Places *int              `yaml:"places,omitempty" json:"places,omitempty"`
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
