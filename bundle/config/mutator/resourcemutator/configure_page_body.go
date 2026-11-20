package resourcemutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

const pageBodyFieldName = "body"

type configurePageBody struct{}

// ConfigurePageBody loads the Markdown body of a page from the file at file_path.
func ConfigurePageBody() bundle.Mutator {
	return &configurePageBody{}
}

func (c configurePageBody) Name() string {
	return "ConfigurePageBody"
}

func (c configurePageBody) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("pages"),
		dyn.AnyKey(),
	)

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			filePath, hasFilePath := v.Get(filePathFieldName).AsString()
			if !hasFilePath {
				return v, nil
			}

			// file_path and body are two ways to provide the same content. Accepting
			// both is ambiguous, so reject it instead of silently picking one.
			body := v.Get(pageBodyFieldName)
			if body.IsValid() && body.Kind() != dyn.KindNil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "both file_path and body are set; specify only one",
					Locations: body.Locations(),
				})
				return v, nil
			}

			contents, err := b.SyncRoot.ReadFile(filePath)
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("failed to read page body from file_path %s: %w", filePath, err)
			}

			// Git checks a text file out with CRLF line endings on Windows by default.
			// Without this the same repository would deploy a different body from
			// Windows than from Linux or macOS, and each deploy would undo the other.
			// Line endings carry no meaning in Markdown.
			body = dyn.V(strings.ReplaceAll(string(contents), "\r\n", "\n"))
			return dyn.Set(v, pageBodyFieldName, body)
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
