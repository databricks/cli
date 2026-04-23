package debug_test

import (
	"context"

	"github.com/databricks/cli/cmd/ucm/utils"
	ucmpkg "github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

// init pre-seeds Ucm.CurrentUser on every loaded deployment so the
// network-backed PopulateCurrentUser mutator short-circuits in unit tests.
// Mirrors the seed installed by cmd/ucm/helpers_test.go.
func init() {
	utils.PreMutateHook = func(_ context.Context, u *ucmpkg.Ucm) {
		if u == nil {
			return
		}
		if u.CurrentUser == nil {
			u.CurrentUser = &config.User{
				ShortName: "test-user",
				User:      &iam.User{UserName: "test-user@example.com"},
			}
		}
	}
}
