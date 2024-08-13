package auth

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestSetHostDoesNotFailWithNoDatabrickscfg(t *testing.T) {
	ctx := context.Background()
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./imaginary-file/databrickscfg")
	err := setHostAndAccountId(ctx, "foo", &auth.PersistentAuth{Host: "test"}, []string{})
	assert.NoError(t, err)
}

func TestSetHost(t *testing.T) {
	ctx := context.Background()
	var persistentAuth auth.PersistentAuth
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")

	// Test error when both flag and argument are provided
	persistentAuth.Host = "val from --host"
	err := setHostAndAccountId(ctx, "profile-1", &persistentAuth, []string{"val from [HOST]"})
	assert.EqualError(t, err, "please only provide a host as an argument or a flag, not both")

	// Test setting host from flag
	persistentAuth.Host = "val from --host"
	err = setHostAndAccountId(ctx, "profile-1", &persistentAuth, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "val from --host", persistentAuth.Host)

	// Test setting host from argument
	persistentAuth.Host = ""
	err = setHostAndAccountId(ctx, "profile-1", &persistentAuth, []string{"val from [HOST]"})
	assert.NoError(t, err)
	assert.Equal(t, "val from [HOST]", persistentAuth.Host)

	// Test setting host from profile
	persistentAuth.Host = ""
	err = setHostAndAccountId(ctx, "profile-1", &persistentAuth, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host1.com", persistentAuth.Host)

	// Test setting host from profile
	persistentAuth.Host = ""
	err = setHostAndAccountId(ctx, "profile-2", &persistentAuth, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host2.com", persistentAuth.Host)
}
