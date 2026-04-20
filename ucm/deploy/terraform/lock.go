package terraform

import (
	"context"
	"fmt"

	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/lock"
)

// defaultLockerFactory is the production factory — builds a Locker backed by
// a local-disk filer under <root>/.databricks/ucm/<target>/state. Using the
// local-disk filer for M1 keeps U5 landable ahead of U4 (which brings the
// workspace-files state backend). When U4 lands, callers wanting remote
// locking can override lockerFactory to point at a workspace-files filer.
func defaultLockerFactory(ctx context.Context, u *ucm.Ucm, user string) (*lock.Locker, error) {
	_, lockDir := lockIdentity(ctx, u)
	f, err := libsfiler.NewLocalClient(lockDir)
	if err != nil {
		return nil, fmt.Errorf("init local-disk filer at %s: %w", lockDir, err)
	}
	return lock.NewLockerWithFiler(user, lockDir, f), nil
}
