package mutator

import (
	"context"
	"reflect"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type dropEmptyStrings struct{}

// DropEmptyStrings removes empty-string values on omitempty resource fields, so
// they are not force-sent to the backend. An empty string reaches this point
// either literally (policy_id: "") or via a variable that resolved to "", so
// this must run after variable resolution.
//
// An explicitly-set zero value is force-sent, defeating the omitempty tag.
// Dropping it here fixes it and makes the result visible in `bundle validate -o json`.
func DropEmptyStrings() bundle.Mutator {
	return &dropEmptyStrings{}
}

func (m *dropEmptyStrings) Name() string {
	return "DropEmptyStrings"
}

func (m *dropEmptyStrings) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	resourcesPath := structpath.NewStringKey(nil, "resources")

	// Drop the empty strings on all struct fields except the ones that must send their zero value.
	var empty []*structpath.PathNode
	collectEmptyStrings(b.Config.View().Get("resources"), resourcesPath, &empty)
	for _, p := range empty {
		if err := b.Config.Delete(p); err != nil {
			return diag.FromErr(err)
		}
	}

	// Keep an empty apps description instead of dropping it: an app update without
	// the field leaves the remote description unchanged, so removing description
	// from config would never converge. Force-sending "" clears it.
	appsPath := structpath.NewStringKey(resourcesPath, "apps")
	var apps []string
	for name, app := range b.Config.View().Lookup(appsPath).MapItems() {
		if !app.Get("description").IsValid() {
			apps = append(apps, name)
		}
	}
	for _, name := range apps {
		err := b.Config.Set(structpath.NewPath(appsPath, name, "description"), "")
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

// collectEmptyStrings appends to out the paths of the struct fields in v that are empty strings,
// unless the field is tagged without omitempty (i.e. its zero value must be sent).
func collectEmptyStrings(v structvar.View, p *structpath.PathNode, out *[]*structpath.PathNode) {
	switch v.Kind() {
	case structvar.KindSequence:
		for i, e := range v.Sequence() {
			collectEmptyStrings(e, structpath.NewIndex(p, i), out)
		}
	case structvar.KindMap:
		r := v.Reflect()
		for r.Kind() == reflect.Pointer {
			r = r.Elem()
		}

		var info *structvar.StructInfo
		switch {
		case r.Kind() == reflect.Struct && !structvar.IsSDKNativeType(r.Type()):
			i := structvar.GetStructInfo(r.Type())
			info = &i
		case r.Kind() == reflect.Map:
		default:
			return
		}

		for k, c := range v.MapItems() {
			if info != nil && !info.ForceEmpty[k] {
				if _, ok := info.Fields[k]; ok {
					if s, ok := c.AsString(); ok && s == "" {
						*out = append(*out, structpath.NewStringKey(p, k))
						continue
					}
				}
			}
			collectEmptyStrings(c, structpath.NewStringKey(p, k), out)
		}
	default:
	}
}
