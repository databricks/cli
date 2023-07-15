package sync

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

func TestPathToRepoPath(t *testing.T) {
	me := iam.User{
		UserName: "jane@doe.com",
	}

	assert.Equal(t, "/Repos/jane@doe.com/foo", repoPathForPath(&me, "/Repos/jane@doe.com/foo/bar/qux"))
	assert.Equal(t, "/Repos/jane@doe.com/foo", repoPathForPath(&me, "/Repos/jane@doe.com/foo/bar"))
	assert.Equal(t, "/Repos/jane@doe.com/foo", repoPathForPath(&me, "/Repos/jane@doe.com/foo"))

	// We expect this function to be called with a path nested under the user's repo path.
	// If this is not the case it should return the input verbatim (albeit cleaned).
	assert.Equal(t, "/Repos/jane@doe.com", repoPathForPath(&me, "/Repos/jane@doe.com"))
	assert.Equal(t, "/Repos/hello@world.com/foo/bar/qux", repoPathForPath(&me, "/Repos/hello@world.com/foo/bar/qux"))
	assert.Equal(t, "/Repos/hello@world.com/foo/bar/qux", repoPathForPath(&me, "/Repos/hello@world.com/foo/bar/qux/."))
}
