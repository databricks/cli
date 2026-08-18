package validate

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// errorForInvalidIdentifiers rejects empty, blank, or control-character values on the
// identifier fields listed in generated.RequiredFields. The backend rejects all three
// with a 400, so failing here avoids a partial deploy.
func errorForInvalidIdentifiers(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	trie := &dyn.TrieNode{}
	for k := range generated.RequiredFields {
		pattern, err := dyn.NewPatternFromString(k)
		if err != nil {
			return diag.FromErr(fmt.Errorf("invalid pattern %q for identifier validation: %w", k, err))
		}
		if err := trie.Insert(pattern); err != nil {
			return diag.FromErr(fmt.Errorf("failed to insert pattern %q into trie: %w", k, err))
		}
	}

	err := dyn.WalkReadOnly(b.Config.Value(), func(p dyn.Path, v dyn.Value) error {
		pattern, ok := trie.SearchPath(p)
		if !ok {
			return nil
		}
		for _, field := range generated.RequiredFields[pattern.String()] {
			if !isIdentifierField(field) {
				continue
			}
			diags = diags.Extend(identifierFieldDiag(b, p, v, field, missingIdentifierIsError(field)))
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	diags = diags.Extend(errorForRegisteredModelIdentifiers(ctx, b))

	sortDiagnostics(diags)
	return diags
}

// errorForRegisteredModelIdentifiers covers registered_models, which the OpenAPI spec
// does not mark required, so generated.RequiredFields has no entry for them.
func errorForRegisteredModelIdentifiers(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("registered_models"), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			diags = diags.Extend(identifierFieldDiag(b, p, v, "name", true))
			diags = diags.Extend(identifierFieldDiag(b, p, v, "catalog_name", false))
			diags = diags.Extend(identifierFieldDiag(b, p, v, "schema_name", false))
			return v, nil
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

// errorForIncompletePipelineLibraries rejects file, notebook, and glob entries without paths.
func errorForIncompletePipelineLibraries(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for key, pipeline := range b.Config.Resources.Pipelines {
		for i, lib := range pipeline.Libraries {
			base := dyn.NewPath(
				dyn.Key("resources"),
				dyn.Key("pipelines"),
				dyn.Key(key),
				dyn.Key("libraries"),
				dyn.Index(i),
			)
			if lib.File != nil && strings.TrimSpace(lib.File.Path) == "" {
				diags = diags.Append(libraryPathDiag(b, base, "file", "pipeline library file path is required"))
			}
			if lib.Notebook != nil && strings.TrimSpace(lib.Notebook.Path) == "" {
				diags = diags.Append(libraryPathDiag(b, base, "notebook", "pipeline library notebook path is required"))
			}
			if lib.Glob != nil && strings.TrimSpace(lib.Glob.Include) == "" {
				diags = diags.Append(libraryPathDiag(b, base, "glob", "pipeline library glob include is required"))
			}
		}
	}

	sortDiagnostics(diags)
	return diags
}

func libraryPathDiag(b *bundle.Bundle, base dyn.Path, field, summary string) diag.Diagnostic {
	fieldPath := base.Append(dyn.Key(field))
	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   summary,
		Locations: locationsFor(b, fieldPath, base),
		Paths:     []dyn.Path{fieldPath},
	}
}

// locationsFor resolves the location of path, falling back to fallback when the field
// carries none of its own (an omitted field has no location to point at).
func locationsFor(b *bundle.Bundle, path, fallback dyn.Path) []dyn.Location {
	if v, err := dyn.GetByPath(b.Config.Value(), path); err == nil && len(v.Locations()) > 0 {
		return v.Locations()
	}
	v, err := dyn.GetByPath(b.Config.Value(), fallback)
	if err != nil {
		return nil
	}
	return v.Locations()
}

func identifierFieldDiag(b *bundle.Bundle, resourcePath dyn.Path, resource dyn.Value, field string, missingIsError bool) diag.Diagnostics {
	vv := resource.Get(field)
	switch vv.Kind() {
	case dyn.KindInvalid, dyn.KindNil:
		if !missingIsError {
			return nil
		}
		return identifierDiagAt(b, resourcePath, field, "is required", "")
	case dyn.KindString:
		reason, detail := invalidIdentifierReason(vv.MustString())
		if reason == "" {
			return nil
		}
		return identifierDiagAt(b, resourcePath, field, reason, detail)
	default:
		return nil
	}
}

func identifierDiagAt(b *bundle.Bundle, resourcePath dyn.Path, field, reason, detail string) diag.Diagnostics {
	fieldPath := slices.Clone(resourcePath).Append(dyn.Key(field))
	return diag.Diagnostics{{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("%s %s %s", resourceSingularName(resourcePath), field, reason),
		Detail:    detail,
		Locations: locationsFor(b, fieldPath, resourcePath),
		Paths:     []dyn.Path{fieldPath},
	}}
}

func invalidIdentifierReason(value string) (reason, detail string) {
	if value == "" {
		return "is required", ""
	}
	if i := strings.IndexFunc(value, unicode.IsControl); i >= 0 {
		r, _ := utf8.DecodeRuneInString(value[i:])
		return "must not contain control characters", fmt.Sprintf("%U at byte offset %d", r, i)
	}
	if strings.TrimSpace(value) == "" {
		return "must not be blank", ""
	}
	return "", ""
}

func isIdentifierField(field string) bool {
	return field == "name" || strings.HasSuffix(field, "_name")
}

// missingIdentifierIsError reports whether an omitted identifier is an error rather than
// a warning. Only the resource's own name is required; UC parents and other *_name
// references may be filled in elsewhere, so those keep warning when omitted.
func missingIdentifierIsError(field string) bool {
	switch field {
	case "name", "display_name", "instance_pool_name":
		return true
	default:
		return false
	}
}

func resourceSingularName(path dyn.Path) string {
	if len(path) >= 2 && path[0].Key() == "resources" {
		plural := path[1].Key()
		if desc, ok := config.SupportedResources()[plural]; ok && desc.SingularName != "" {
			return desc.SingularName
		}
		return plural
	}
	return "resource"
}
