package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/walk"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
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
	// Drop the empty strings on all struct fields except the ones that must send their zero value.
	var empty []*structpath.PathNode
	walk.ResourceStrings(&b.Config, walk.EmptyFields, func(n *walk.Node, _ string) (string, bool) {
		empty = append(empty, n.Path())
		return "", false
	})
	for _, p := range empty {
		if err := b.Config.Delete(p); err != nil {
			return diag.FromErr(err)
		}
	}

	// Keep an empty apps description instead of dropping it: an app update without
	// the field leaves the remote description unchanged, so removing description
	// from config would never converge. Force-sending "" clears it.
	for name, app := range b.Config.Resources.Apps {
		if app != nil && (app.Description != "" || walk.Forced(app.ForceSendFields, "Description")) {
			continue
		}
		err := b.Config.Set(structpath.NewPath(nil, "resources", "apps", name, "description"), "")
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}
