package skills

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/log"
)

// GroupSkills returns the names of the local skills whose Commands include
// group, such as "fs" or "bundle deployment" (see SkillMeta.Commands). It
// returns nil when hints are off, when no local catalog is set, or when the
// catalog cannot be read. It reads only the local catalog, so help never makes
// a network call.
func GroupSkills(ctx context.Context, group string) []string {
	if !HintsEnabled(ctx) {
		return nil
	}

	// A folder that cannot be read leaves help without a footer. Help must still print.
	catalog, err := LocalCatalog(ctx)
	if err != nil {
		log.Debugf(ctx, "skills hint: %s", err)
		return nil
	}
	if catalog == nil {
		return nil
	}

	var names []string
	for _, s := range catalog.Skills {
		if slices.Contains(s.Commands, group) {
			names = append(names, s.Name)
		}
	}
	return names
}

// WriteGroupHint writes the help footer for the command group named group.
// The footer names each local skill that GroupSkills returns for group. It
// writes nothing when GroupSkills is empty.
func WriteGroupHint(ctx context.Context, w io.Writer, group string) {
	names := GroupSkills(ctx, group)
	if len(names) == 0 {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Coding agents: read the skills for this command group before you use it.")
	for _, name := range names {
		fmt.Fprintf(w, "  %s\n", hintReadCommand(name))
	}
}

// hintReadCommand returns the command that reads the skill named name. A name
// that starts with "-" gets "--" first, so Cobra does not read it as a flag.
func hintReadCommand(name string) string {
	if strings.HasPrefix(name, "-") {
		return "databricks skills read -- " + name
	}
	return "databricks skills read " + name
}
