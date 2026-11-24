package resourcemutator

import (
	"context"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type pageFixups struct{}

// PageFixups normalizes the synonyms of a page the way the API stores them: trimmed,
// deduplicated and sorted. Otherwise the list would differ from what the API returns and
// show as changed on every plan.
func PageFixups() bundle.Mutator {
	return &pageFixups{}
}

func (m *pageFixups) Name() string {
	return "PageFixups"
}

func (m *pageFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, page := range b.Config.Resources.Pages {
		if page == nil || page.Synonyms == nil {
			continue
		}

		synonyms := make([]string, 0, len(page.Synonyms))
		for _, synonym := range page.Synonyms {
			synonyms = append(synonyms, strings.TrimSpace(synonym))
		}
		slices.Sort(synonyms)
		page.Synonyms = slices.Compact(synonyms)
	}

	return nil
}
