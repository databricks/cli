package mutator

import (
	"context"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

// Temporary: this mutator collects telemetry on escape patterns ($${}, $$, \${}, \$)
// in bundle configs. It will be removed once we align on the escape syntax.
type collectEscapeTelemetry struct{}

func CollectEscapeTelemetry() bundle.Mutator {
	return &collectEscapeTelemetry{}
}

func (*collectEscapeTelemetry) Name() string {
	return "CollectEscapeTelemetry"
}

func (*collectEscapeTelemetry) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var hasDoubleDollarBrace, hasDoubleDollar, hasBackslashDollarBrace, hasBackslashDollar bool

	walkStrings(reflect.ValueOf(&b.Config), func(s string) {
		if !hasDoubleDollarBrace && strings.Contains(s, "$${") {
			hasDoubleDollarBrace = true
		}
		if !hasDoubleDollar && containsDoubleDollarWithoutBrace(s) {
			hasDoubleDollar = true
		}
		if !hasBackslashDollarBrace && strings.Contains(s, "\\${") {
			hasBackslashDollarBrace = true
		}
		if !hasBackslashDollar && containsBackslashDollarWithoutBrace(s) {
			hasBackslashDollar = true
		}
	})

	if hasDoubleDollarBrace {
		b.Metrics.SetBoolValue("config_has_double_dollar_brace", true)
	}
	if hasDoubleDollar {
		b.Metrics.SetBoolValue("config_has_double_dollar", true)
	}
	if hasBackslashDollarBrace {
		b.Metrics.SetBoolValue("config_has_backslash_dollar_brace", true)
	}
	if hasBackslashDollar {
		b.Metrics.SetBoolValue("config_has_backslash_dollar", true)
	}

	return nil
}

// walkStrings calls fn for every string value in the configuration value v,
// including strings stored in interface values (e.g. variable defaults).
func walkStrings(v reflect.Value, fn func(string)) {
	switch v.Kind() {
	case reflect.String:
		fn(v.String())
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			walkStrings(v.Elem(), fn)
		}
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			f := t.Field(i)
			// Like the configuration conversion, skip fields without a JSON name.
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if !f.IsExported() || (!f.Anonymous && (name == "" || name == "-")) {
				continue
			}
			walkStrings(v.Field(i), fn)
		}
	case reflect.Slice:
		for i := range v.Len() {
			walkStrings(v.Index(i), fn)
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			walkStrings(iter.Value(), fn)
		}
	default:
	}
}

// containsDoubleDollarWithoutBrace returns true if s contains "$$" not followed by "{".
func containsDoubleDollarWithoutBrace(s string) bool {
	for i := range len(s) - 1 {
		if s[i] == '$' && s[i+1] == '$' {
			if i+2 >= len(s) || s[i+2] != '{' {
				return true
			}
		}
	}
	return false
}

// containsBackslashDollarWithoutBrace returns true if s contains "\$" not followed by "{".
func containsBackslashDollarWithoutBrace(s string) bool {
	for i := range len(s) - 1 {
		if s[i] == '\\' && s[i+1] == '$' {
			if i+2 >= len(s) || s[i+2] != '{' {
				return true
			}
		}
	}
	return false
}
