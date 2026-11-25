package resourcemutator

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type captureUCDependencies struct{}

// If a user defines a UC schema in the bundle, other resources can refer to it using the
// `${resources.schemas.<schema_key>.name}` syntax. Using this syntax allows
// TF to capture the deploy time dependency this resource has on the schema and deploy changes
// to the schema before deploying the dependent resource.
//
// Similarly, if a user defines a UC catalog in the bundle, other resources can refer to it using
// the `${resources.catalogs.<catalog_key>.name}` syntax. This captures the deploy time
// dependency the resource has on the catalog.
//
// This mutator translates any implicit catalog or schema references to the explicit syntax.
// Resource types declare the fields to translate by implementing [resources.UCNameDeclarer].
func CaptureUCDependencies() bundle.Mutator {
	return &captureUCDependencies{}
}

func (m *captureUCDependencies) Name() string {
	return "CaptureUCDependencies"
}

type ucSchemaKey struct {
	catalogName string
	schemaName  string
}

// ucTargets maps the names of the catalogs and schemas defined in the bundle to
// references to them. It is built before any name is rewritten, because a schema's
// own catalog name is rewritten too while other resources still match against it.
type ucTargets struct {
	catalogs map[string]string
	schemas  map[ucSchemaKey]string
}

func newUCTargets(groups []config.ResourceGroup) *ucTargets {
	t := &ucTargets{
		catalogs: map[string]string{},
		schemas:  map[ucSchemaKey]string{},
	}
	for _, group := range groups {
		// Sorted, so that the first key wins if several resources define the same name.
		for _, key := range slices.Sorted(maps.Keys(group.Resources)) {
			r := group.Resources[key]
			if reflect.ValueOf(r).IsNil() {
				continue
			}
			ref := fmt.Sprintf("${resources.%s.%s.name}", group.Description.PluralName, key)
			if c, ok := r.(resources.UCCatalogResource); ok {
				if name := c.UCCatalogName(); name != "" {
					addIfAbsent(t.catalogs, name, ref)
				}
			}
			if s, ok := r.(resources.UCSchemaResource); ok {
				if catalogName, schemaName := s.UCSchemaName(); catalogName != "" && schemaName != "" {
					addIfAbsent(t.schemas, ucSchemaKey{catalogName, schemaName}, ref)
				}
			}
		}
	}
	return t
}

func addIfAbsent[K comparable](m map[K]string, key K, value string) {
	if _, ok := m[key]; !ok {
		m[key] = value
	}
}

// resolveSchema returns the explicit schema reference if the given catalogName
// and schemaName match a schema defined in the bundle. Otherwise returns schemaName
// unchanged.
func (t *ucTargets) resolveSchema(catalogName, schemaName string) string {
	if ref, ok := t.schemas[ucSchemaKey{catalogName, schemaName}]; ok {
		return ref
	}
	return schemaName
}

// resolveCatalog returns the explicit catalog reference if catalogName matches
// a catalog defined in the bundle. Otherwise returns catalogName unchanged.
func (t *ucTargets) resolveCatalog(catalogName string) string {
	if ref, ok := t.catalogs[catalogName]; ok {
		return ref
	}
	return catalogName
}

// splitUCName splits a compound UC identifier into exactly n dot-separated
// components, returning false if it has a different number of components. Callers
// skip names containing a ${...} reference beforehand, so this is a plain split:
// a reference would push the component count past n and be rejected here anyway.
func splitUCName(name string, n int) ([]string, bool) {
	parts := strings.Split(name, ".")
	if len(parts) != n {
		return nil, false
	}
	return parts, true
}

// resolveCompound applies resolveCatalog and resolveSchema to the catalog and
// schema of a compound UC name: prefix followed by n dot-separated components, the
// first two being a catalog and a schema. A name without the prefix or with a
// different number of components is returned unchanged.
func (t *ucTargets) resolveCompound(name, prefix string, n int) string {
	// A name that already contains a reference is left as is: we only rewrite a
	// fully literal name and do not support a mix of references and literals.
	if dynvar.ContainsVariableReference(name) {
		return name
	}
	rest, ok := strings.CutPrefix(name, prefix)
	if !ok {
		return name
	}
	parts, ok := splitUCName(rest, n)
	if !ok {
		return name
	}
	catalogName, schemaName := parts[0], parts[1]
	parts[0] = t.resolveCatalog(catalogName)
	parts[1] = t.resolveSchema(catalogName, schemaName)
	return prefix + strings.Join(parts, ".")
}

func (t *ucTargets) resolve(name resources.UCName) {
	if name.Compound != nil {
		*name.Compound = t.resolveCompound(*name.Compound, name.Prefix, name.Parts)
		return
	}
	// Schemas are resolved against the catalog name before it is rewritten.
	for _, schema := range name.Schemas {
		*schema = t.resolveSchema(*name.Catalog, *schema)
	}
	*name.Catalog = t.resolveCatalog(*name.Catalog)
}

func (m *captureUCDependencies) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	groups := b.Config.Resources.AllResources()
	targets := newUCTargets(groups)
	for _, group := range groups {
		for _, r := range group.Resources {
			d, ok := r.(resources.UCNameDeclarer)
			if !ok || reflect.ValueOf(r).IsNil() {
				continue
			}
			for _, name := range d.UCNames() {
				targets.resolve(name)
			}
		}
	}
	return nil
}
