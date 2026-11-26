package skills

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/databricks/cli/libs/log"
)

// WriteGroupHint writes the help footer for the command group named group,
// such as "fs". The footer names each local skill whose Commands include
// group. It writes nothing when hints are off, when no skill maps the group, or
// when the local catalog cannot be read. It reads only the local catalog, so
// help never makes a network call.
func WriteGroupHint(ctx context.Context, w io.Writer, group string) {
	if !HintsEnabled(ctx) {
		return
	}

	// A folder that cannot be read leaves help without a footer. Help must still print.
	catalog, err := LocalCatalog(ctx)
	if err != nil {
		log.Debugf(ctx, "skills hint: %s", err)
		return
	}
	if catalog == nil {
		return
	}

	var names []string
	for _, s := range catalog.Skills {
		if slices.Contains(s.Commands, group) {
			names = append(names, s.Name)
		}
	}
	if len(names) == 0 {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Coding agents: read the skills for this command group before you use it.")
	for _, name := range names {
		fmt.Fprintf(w, "  databricks skills read %s\n", name)
	}
}
