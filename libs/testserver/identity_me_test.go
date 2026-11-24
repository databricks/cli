package testserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIdentityMeMatchesMeUser checks that the fake answers identity/me and SCIM /Me with the same id and
// userName for each kind of principal that the id check in the acceptance tests can run as. The
// assumed-group token is the exception by design: identity/me names the group, while SCIM /Me still
// names the user behind it.
func TestIdentityMeMatchesMeUser(t *testing.T) {
	s := NewFakeWorkspace("", "")
	for _, tc := range []struct {
		name, token, idField, nameField string
	}{
		{name: "user", token: "dapi-user", idField: "user_id", nameField: "username"},
		{name: "service principal", token: ServicePrincipalTokenPrefix + "-sp", idField: "service_principal_id", nameField: "application_id"},
		{name: "guest service principal", token: GuestServicePrincipalTokenPrefix + "-guest", idField: "service_principal_id", nameField: "application_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			me := s.MeUser(tc.token)
			body, ok := s.IdentityMe(tc.token).(map[string]any)
			require.True(t, ok)
			var principal map[string]any
			for _, key := range []string{"user", "service_principal"} {
				if p, found := body[key].(map[string]any); found {
					principal = p
				}
			}
			require.NotNil(t, principal, "identity/me must describe a user or a service principal")
			assert.Equal(t, me.Id, principal[tc.idField])
			assert.Equal(t, me.UserName, principal[tc.nameField])
		})
	}

	t.Run("assumed group", func(t *testing.T) {
		token := AssumedGroupTokenPrefix + "-group"
		body, ok := s.IdentityMe(token).(map[string]any)
		require.True(t, ok)
		group, ok := body["group"].(map[string]any)
		require.True(t, ok, "identity/me describes the assumed group, not the user")
		assert.Equal(t, AssumedGroupId, group["group_id"])
	})
}
