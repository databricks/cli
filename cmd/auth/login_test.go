package auth

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetHostDoesNotFailWithNoDatabrickscfg(t *testing.T) {
	ctx := context.Background()
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./imaginary-file/databrickscfg")
	err := setHostAndAccountId(ctx, "foo", &auth.PersistentAuth{Host: "test"}, []string{})
	assert.NoError(t, err)
}

func TestSetHost(t *testing.T) {
	var persistentAuth auth.PersistentAuth
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(context.Background())

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

	// Test host is not set. Should prompt.
	persistentAuth.Host = ""
	err = setHostAndAccountId(ctx, "", &persistentAuth, []string{})
	assert.EqualError(t, err, "the command is being run in a non-interactive environment, please specify a host using --host")
}

func TestSetAccountId(t *testing.T) {
	var persistentAuth auth.PersistentAuth
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(context.Background())

	// Test setting account-id from flag
	persistentAuth.AccountID = "val from --account-id"
	err := setHostAndAccountId(ctx, "account-profile", &persistentAuth, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://accounts.cloud.databricks.com", persistentAuth.Host)
	assert.Equal(t, "val from --account-id", persistentAuth.AccountID)

	// Test setting account_id from profile
	persistentAuth.AccountID = ""
	err = setHostAndAccountId(ctx, "account-profile", &persistentAuth, []string{})
	require.NoError(t, err)
	assert.Equal(t, "https://accounts.cloud.databricks.com", persistentAuth.Host)
	assert.Equal(t, "id-from-profile", persistentAuth.AccountID)

	// Neither flag nor profile account-id is set, should prompt
	persistentAuth.AccountID = ""
	persistentAuth.Host = "https://accounts.cloud.databricks.com"
	err = setHostAndAccountId(ctx, "", &persistentAuth, []string{})
	assert.EqualError(t, err, "the command is being run in a non-interactive environment, please specify an account ID using --account-id")
}
