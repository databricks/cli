package metricviews

import (
	"strings"
	"unicode"
)

// Normalize rewrites the metric view into the canonical form the backend
// produces, so a locally-authored definition and the backend's stored
// view_text compare equal under structdiff (i.e. it removes drift that is
// purely an artifact of backend normalization).
//
// The rules are derived empirically from a CREATE-then-DESCRIBE round-trip
// against a live workspace (see normalize_test.go):
//
//   - Enum-valued fields are lowercased. The backend accepts any casing and
//     canonicalizes to lowercase for every enum except format.type, which it
//     instead REQUIRES to be lowercase (a non-lowercase value fails CREATE);
//     lowercasing it here is therefore both safe and necessary.
//   - Trailing whitespace is trimmed from scalar values. Leading and internal
//     whitespace is preserved, matching the backend.
//   - Everything else is left verbatim: the backend does not reformat SQL
//     expressions, filters, sources, or identifiers, does not canonicalize the
//     version string or data_type casing, and injects no default fields.
//
// Layout-only differences (key order, quoting, flow-vs-block, blank lines) are
// not handled here because typed parsing already absorbs them.
//
// Normalize is idempotent and mutates the receiver in place.
func (m *MetricView) Normalize() {
	switch {
	case m.V10 != nil:
		m.V10.Normalize()
	case m.SingleSource != nil:
		m.SingleSource.Normalize()
	case m.MultiSource != nil:
		m.MultiSource.Normalize()
	}
}

// trimTrailing removes trailing whitespace only; the backend preserves leading
// and internal whitespace in scalar values.
func trimTrailing(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// trimTrailingPtr trims trailing whitespace through a string pointer, leaving nil untouched.
func trimTrailingPtr(s *string) {
	if s != nil {
		*s = trimTrailing(*s)
	}
}

// lowerEnum canonicalizes an enum value to the backend's lowercase form.
func lowerEnum(s string) string {
	return strings.ToLower(trimTrailing(s))
}

// Normalize canonicalizes a v0.1/v1.0 metric view in place.
func (v *MetricViewV10) Normalize() {
	v.Version = trimTrailing(v.Version)
	v.Source = trimTrailing(v.Source)
	trimTrailingPtr(v.Filter)
	for i := range v.Dimensions {
		v.Dimensions[i].normalize()
	}
	for i := range v.Measures {
		v.Measures[i].normalize()
	}
	for i := range v.Joins {
		v.Joins[i].normalize()
	}
	if v.Materialization != nil {
		v.Materialization.normalize()
	}
	for i := range v.Parameters {
		v.Parameters[i].normalize()
	}
}

// Normalize canonicalizes a v1.1 single-source metric view in place.
func (v *SingleSourceMetricView) Normalize() {
	v.Version = trimTrailing(v.Version)
	v.Source = trimTrailing(v.Source)
	trimTrailingPtr(v.Filter)
	trimTrailingPtr(v.Comment)
	for i := range v.Joins {
		v.Joins[i].normalize()
	}
	for i := range v.Parameters {
		v.Parameters[i].normalize()
	}
	for i := range v.Dimensions {
		v.Dimensions[i].normalize()
	}
	for i := range v.Measures {
		v.Measures[i].normalize()
	}
	if v.Materialization != nil {
		v.Materialization.normalize()
	}
}

// Normalize canonicalizes a v1.1 multi-source metric view in place.
func (v *MultiSourceMetricView) Normalize() {
	v.Version = trimTrailing(v.Version)
	// view_type is an exact-match discriminator (UPPER_SNAKE), not a lowercased enum.
	v.ViewType = trimTrailing(v.ViewType)
	for i := range v.Sources {
		v.Sources[i].normalize()
	}
	trimTrailingPtr(v.Comment)
	for i := range v.Parameters {
		v.Parameters[i].normalize()
	}
	for i := range v.Dimensions {
		v.Dimensions[i].normalize()
	}
	for i := range v.Measures {
		v.Measures[i].normalize()
	}
}

func (c *ColumnV10) normalize() {
	c.Name = trimTrailing(c.Name)
	c.Expr = trimTrailing(c.Expr)
	for i := range c.Window {
		c.Window[i].normalize()
	}
}

func (c *ColumnV11) normalize() {
	trimTrailingPtr(c.Name)
	c.Expr = trimTrailing(c.Expr)
	for i := range c.Window {
		c.Window[i].normalize()
	}
	trimTrailingPtr(c.Comment)
	trimTrailingPtr(c.DisplayName)
	if c.Format != nil {
		c.Format.normalize()
	}
	for i := range c.Synonyms {
		c.Synonyms[i] = trimTrailing(c.Synonyms[i])
	}
}

func (w *WindowSpec) normalize() {
	w.Order = trimTrailing(w.Order)
	w.Semiadditive = lowerEnum(w.Semiadditive)
	w.Range = lowerEnum(w.Range)
	w.Offset = trimTrailing(w.Offset)
}

func (f *ColumnFormat) normalize() {
	f.Type = lowerEnum(f.Type)
	if f.DecimalPlaces != nil {
		f.DecimalPlaces.normalize()
	}
	// date_format/time_format/currency_code/abbreviation are free-form values:
	// trim trailing whitespace but do not change their casing.
	trimTrailingPtr(f.Abbreviation)
	trimTrailingPtr(f.CurrencyCode)
	trimTrailingPtr(f.DateFormat)
	trimTrailingPtr(f.TimeFormat)
}

func (d *DecimalPlaces) normalize() {
	d.Type = DecimalPlacesType(lowerEnum(string(d.Type)))
}

func (j *Join) normalize() {
	j.Name = trimTrailing(j.Name)
	j.Source = trimTrailing(j.Source)
	trimTrailingPtr(j.On)
	for i := range j.Using {
		j.Using[i] = trimTrailing(j.Using[i])
	}
	for i := range j.Joins {
		j.Joins[i].normalize()
	}
	if j.Cardinality != nil {
		*j.Cardinality = lowerEnum(*j.Cardinality)
	}
}

func (m *Materialization) normalize() {
	// schedule is a free-form string (e.g. "EVERY 1 HOUR"), stored verbatim.
	m.Schedule = trimTrailing(m.Schedule)
	m.Mode = lowerEnum(m.Mode)
	for i := range m.MaterializedViews {
		m.MaterializedViews[i].normalize()
	}
}

func (m *MaterializedView) normalize() {
	m.Name = trimTrailing(m.Name)
	m.MVType = lowerEnum(m.MVType)
	for i := range m.Dimensions {
		m.Dimensions[i] = trimTrailing(m.Dimensions[i])
	}
	for i := range m.Measures {
		m.Measures[i] = trimTrailing(m.Measures[i])
	}
}

func (p *ParameterV11) normalize() {
	p.Name = trimTrailing(p.Name)
	// data_type casing is preserved by the backend; do not lowercase it.
	p.DataType = trimTrailing(p.DataType)
	p.Default.Expr = trimTrailing(p.Default.Expr)
}

func (p *ParameterV10) normalize() {
	p.Name = trimTrailing(p.Name)
	p.DataType = trimTrailing(p.DataType)
	trimTrailingPtr(p.Default)
}

func (s *SourceNode) normalize() {
	s.Name = trimTrailing(s.Name)
	s.From = trimTrailing(s.From)
	for i := range s.PrimaryKey {
		s.PrimaryKey[i] = trimTrailing(s.PrimaryKey[i])
	}
	for i := range s.Relationships {
		s.Relationships[i].normalize()
	}
}

func (r *Relationship) normalize() {
	r.RefSource = trimTrailing(r.RefSource)
	for i := range r.ForeignKey {
		r.ForeignKey[i] = trimTrailing(r.ForeignKey[i])
	}
	trimTrailingPtr(r.On)
}
