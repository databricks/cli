package metricviews

import (
	"errors"
	"fmt"
	"strings"
)

// Validate checks a decoded metric view before it is used as input.
// Loading persisted state should use Parse without calling Validate.
func Validate(m *MetricView) error {
	if m == nil {
		return errors.New("metricviews: missing metric view")
	}
	variants := 0
	if m.V10 != nil {
		variants++
	}
	if m.SingleSource != nil {
		variants++
	}
	if m.MultiSource != nil {
		variants++
	}
	if variants != 1 {
		return errors.New("metricviews: exactly one view variant is required")
	}

	switch {
	case m.V10 != nil:
		if m.V10.Version != version01 && m.V10.Version != version10 {
			return fmt.Errorf("metricviews: invalid YAML version: %q", m.V10.Version)
		}
		if err := validateText("source", m.V10.Source); err != nil {
			return err
		}
		for i := range m.V10.Dimensions {
			if err := validateColumnV10Model(&m.V10.Dimensions[i]); err != nil {
				return fmt.Errorf("dimensions[%d]: %w", i, err)
			}
		}
		for i := range m.V10.Measures {
			if err := validateColumnV10Model(&m.V10.Measures[i]); err != nil {
				return fmt.Errorf("measures[%d]: %w", i, err)
			}
		}
		if err := validateJoins(m.V10.Joins); err != nil {
			return err
		}
		for i := range m.V10.Parameters {
			if err := validateParameter(m.V10.Parameters[i].Name, m.V10.Parameters[i].DataType); err != nil {
				return fmt.Errorf("parameters[%d]: %w", i, err)
			}
		}
		return validateMaterializationModel(m.V10.Materialization)
	case m.SingleSource != nil:
		v := m.SingleSource
		if v.Version != version11 {
			return fmt.Errorf("metricviews: invalid YAML version: %q", v.Version)
		}
		if v.ViewType != "" && v.ViewType != viewTypeSingleSource {
			return fmt.Errorf("metricviews: unsupported v1.1 view_type %q", v.ViewType)
		}
		if err := validateText("source", v.Source); err != nil {
			return err
		}
		for i := range v.Dimensions {
			if err := validateColumnV11Model(&v.Dimensions[i]); err != nil {
				return fmt.Errorf("dimensions[%d]: %w", i, err)
			}
		}
		for i := range v.Measures {
			if err := validateColumnV11Model(&v.Measures[i]); err != nil {
				return fmt.Errorf("measures[%d]: %w", i, err)
			}
		}
		if err := validateJoins(v.Joins); err != nil {
			return err
		}
		if err := validateParametersV11(v.Parameters); err != nil {
			return err
		}
		return validateMaterializationModel(v.Materialization)
	default:
		v := m.MultiSource
		if v.Version != version11 {
			return fmt.Errorf("metricviews: invalid YAML version: %q", v.Version)
		}
		if v.ViewType != viewTypeMultiSource {
			return fmt.Errorf("metricviews: view_type must be %s", viewTypeMultiSource)
		}
		if len(v.Sources) == 0 {
			return errors.New("sources are required")
		}
		for i := range v.Sources {
			s := &v.Sources[i]
			if err := validateText("name", s.Name); err != nil {
				return fmt.Errorf("sources[%d]: %w", i, err)
			}
			if err := validateText("from", s.From); err != nil {
				return fmt.Errorf("sources[%d]: %w", i, err)
			}
			for j := range s.Relationships {
				if err := validateText("ref", s.Relationships[j].RefSource); err != nil {
					return fmt.Errorf("sources[%d].relationships[%d]: %w", i, j, err)
				}
			}
		}
		for i := range v.Dimensions {
			if err := validateColumnV11Model(&v.Dimensions[i]); err != nil {
				return fmt.Errorf("dimensions[%d]: %w", i, err)
			}
		}
		for i := range v.Measures {
			if err := validateColumnV11Model(&v.Measures[i]); err != nil {
				return fmt.Errorf("measures[%d]: %w", i, err)
			}
		}
		return validateParametersV11(v.Parameters)
	}
}

func validateText(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

func validateColumnV10Model(c *ColumnV10) error {
	if err := validateText("name", c.Name); err != nil {
		return err
	}
	if err := validateText("expr", c.Expr); err != nil {
		return err
	}
	return validateColumnDetailsModel(c.Window, c.Format)
}

func validateColumnV11Model(c *ColumnV11) error {
	if c.Name != nil {
		if err := validateText("name", *c.Name); err != nil {
			return err
		}
	}
	if err := validateText("expr", c.Expr); err != nil {
		return err
	}
	return validateColumnDetailsModel(c.Window, c.Format)
}

func validateColumnDetailsModel(windows []WindowSpec, format *ColumnFormat) error {
	for i := range windows {
		w := &windows[i]
		for _, field := range []struct{ name, value string }{{"order", w.Order}, {"semiadditive", w.Semiadditive}, {"range", w.Range}} {
			if err := validateText(field.name, field.value); err != nil {
				return fmt.Errorf("window[%d]: %w", i, err)
			}
		}
	}
	if format != nil {
		if err := validateFormatModel(format); err != nil {
			return fmt.Errorf("format: %w", err)
		}
	}
	return nil
}

func validateJoins(joins []Join) error {
	for i := range joins {
		if err := validateText("name", joins[i].Name); err != nil {
			return fmt.Errorf("joins[%d]: %w", i, err)
		}
		if err := validateText("source", joins[i].Source); err != nil {
			return fmt.Errorf("joins[%d]: %w", i, err)
		}
		if err := validateJoins(joins[i].Joins); err != nil {
			return fmt.Errorf("joins[%d]: %w", i, err)
		}
	}
	return nil
}

func validateParameter(name, dataType string) error {
	if err := validateText("name", name); err != nil {
		return err
	}
	return validateText("data_type", dataType)
}

func validateParametersV11(parameters []ParameterV11) error {
	for i := range parameters {
		if err := validateParameter(parameters[i].Name, parameters[i].DataType); err != nil {
			return fmt.Errorf("parameters[%d]: %w", i, err)
		}
	}
	return nil
}

func validateMaterializationModel(m *Materialization) error {
	if m == nil {
		return nil
	}
	if err := validateText("materialization.schedule", m.Schedule); err != nil {
		return err
	}
	if err := validateText("materialization.mode", m.Mode); err != nil {
		return err
	}
	if len(m.MaterializedViews) == 0 {
		return errors.New("materialization.materialized_views are required")
	}
	for i := range m.MaterializedViews {
		mv := &m.MaterializedViews[i]
		if err := validateText("name", mv.Name); err != nil {
			return fmt.Errorf("materialization.materialized_views[%d]: %w", i, err)
		}
		if err := validateText("type", mv.MVType); err != nil {
			return fmt.Errorf("materialization.materialized_views[%d]: %w", i, err)
		}
	}
	return nil
}

func validateFormatModel(f *ColumnFormat) error {
	allowed, err := formatAllowedFields(f.Type)
	if err != nil {
		return err
	}
	for _, field := range []struct {
		name    string
		present bool
	}{
		{"decimal_places", f.DecimalPlaces != nil},
		{"hide_group_separator", f.HideGroupSeparator != nil},
		{"abbreviation", f.Abbreviation != nil},
		{"currency_code", f.CurrencyCode != nil},
		{"date_format", f.DateFormat != nil},
		{"time_format", f.TimeFormat != nil},
		{"leading_zeros", f.LeadingZeros != nil},
	} {
		if field.present && !allowed[field.name] {
			return fmt.Errorf("field %q is not valid for format type %q", field.name, f.Type)
		}
	}
	switch f.Type {
	case "currency":
		if f.CurrencyCode == nil {
			return errors.New("currency_code is required")
		}
		if err := validateText("currency_code", *f.CurrencyCode); err != nil {
			return err
		}
	case "date":
		if f.DateFormat == nil {
			return errors.New("date_format is required")
		}
		if err := validateText("date_format", *f.DateFormat); err != nil {
			return err
		}
	case "date_time":
		if f.DateFormat == nil || f.TimeFormat == nil {
			return errors.New("date_format and time_format are required")
		}
		if err := validateText("date_format", *f.DateFormat); err != nil {
			return err
		}
		if err := validateText("time_format", *f.TimeFormat); err != nil {
			return err
		}
	}
	if f.DecimalPlaces != nil {
		switch f.DecimalPlaces.Type {
		case "max", "exact", "all":
		default:
			return fmt.Errorf("unsupported decimal_places.type %q", f.DecimalPlaces.Type)
		}
	}
	return nil
}

func formatAllowedFields(typ string) (map[string]bool, error) {
	allowed := map[string]bool{"type": true}
	switch typ {
	case "number":
		allowed["decimal_places"], allowed["hide_group_separator"], allowed["abbreviation"] = true, true, true
	case "currency":
		allowed["decimal_places"], allowed["hide_group_separator"], allowed["abbreviation"], allowed["currency_code"] = true, true, true, true
	case "percentage", "byte":
		allowed["decimal_places"], allowed["hide_group_separator"] = true, true
	case "date":
		allowed["date_format"], allowed["leading_zeros"] = true, true
	case "date_time":
		allowed["date_format"], allowed["time_format"], allowed["leading_zeros"] = true, true, true
	default:
		return nil, fmt.Errorf("unsupported format type %q", typ)
	}
	return allowed, nil
}
