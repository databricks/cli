package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
)

type defineDefaultTarget struct {
	name string
}

// DefineDefaultTarget adds a target named "default" to the configuration if
// none have been defined. Keeps verb wiring uniform (SelectTarget always has
// something to pick) even when ucm.yml omits `targets:` entirely.
func DefineDefaultTarget() ucm.Mutator {
	return &defineDefaultTarget{name: "default"}
}

func (m *defineDefaultTarget) Name() string {
	return fmt.Sprintf("DefineDefaultTarget(%s)", m.name)
}

func (m *defineDefaultTarget) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	if len(u.Config.Targets) > 0 {
		return nil
	}
	u.Config.Targets = map[string]*config.Target{m.name: {}}
	return nil
}
