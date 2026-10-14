package iamutil

import (
	"testing"

	"github.com/databricks/cli/libs/sdktypes/scim/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetShortUserName(t *testing.T) {
	tests := []struct {
		name     string
		user     *scim.User
		expected string
	}{
		{
			user: &scim.User{
				UserName: new("test.user.1234@example.com"),
			},
			expected: "test_user_1234",
		},
		{
			user: &scim.User{
				UserName: new("tést.üser@example.com"),
			},
			expected: "tést_üser",
		},
		{
			user: &scim.User{
				UserName: new("test$.user@example.com"),
			},
			expected: "test_user",
		},
		{
			user: &scim.User{
				UserName: new(`jöhn.dœ@domain.com`), // Using non-ASCII characters.
			},
			expected: "jöhn_dœ",
		},
		{
			user: &scim.User{
				UserName: new(`first+tag@email.com`), // The plus (+) sign is used for "sub-addressing" in some email services.
			},
			expected: "first_tag",
		},
		{
			user: &scim.User{
				UserName: new(`email@sub.domain.com`), // Using a sub-domain.
			},
			expected: "email",
		},
		{
			user: &scim.User{
				UserName: new(`"_quoted"@domain.com`), // Quoted strings can be part of the local-part.
			},
			expected: "quoted",
		},
		{
			user: &scim.User{
				UserName: new(`name-o'mally@website.org`), // Single quote in the local-part.
			},
			expected: "name_o_mally",
		},
		{
			user: &scim.User{
				UserName: new(`user%domain@external.com`), // Percent sign can be used for email routing in legacy systems.
			},
			expected: "user_domain",
		},
		{
			user: &scim.User{
				UserName: new(`long.name.with.dots@domain.net`), // Multiple dots in the local-part.
			},
			expected: "long_name_with_dots",
		},
		{
			user: &scim.User{
				UserName: new(`me&you@together.com`), // Using an ampersand (&) in the local-part.
			},
			expected: "me_you",
		},
		{
			user: &scim.User{
				UserName: new(`user!def!xyz@domain.org`), // The exclamation mark can be valid in some legacy systems.
			},
			expected: "user_def_xyz",
		},
		{
			user: &scim.User{
				UserName: new(`admin@ιντερνετ.com`), // Domain in non-ASCII characters (IDN or Internationalized Domain Name).
			},
			expected: "admin",
		},
		{
			user: &scim.User{
				UserName:    new(`1706906c-c0a2-4c25-9f57-3a7aa3cb8123`),
				DisplayName: new("my-service-principal"),
			},
			expected: "my_service_principal",
		},
		{
			user: &scim.User{
				UserName: new(`1706906c-c0a2-4c25-9f57-3a7aa3cb8123`),
				// This service princpal has DisplayName (it's an optional property)
			},
			expected: "1706906c_c0a2_4c25_9f57_3a7aa3cb8123",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, GetShortUserName(tt.user))
	}
}

func TestGetShortUserDomainFriendlyName(t *testing.T) {
	tests := []struct {
		name     string
		user     *scim.User
		expected string
	}{
		{
			name: "basic email with dots",
			user: &scim.User{
				UserName: new("test.user.1234@example.com"),
			},
			expected: "test-user-1234",
		},
		{
			name: "email with underscores",
			user: &scim.User{
				UserName: new("test_user@example.com"),
			},
			expected: "test-user",
		},
		{
			name: "email with special characters",
			user: &scim.User{
				UserName: new("test$#%user@example.com"),
			},
			expected: "test-user",
		},
		{
			name: "service principal with display name",
			user: &scim.User{
				UserName:    new(`1706906c-c0a2-4c25-9f57-3a7aa3cb8123`),
				DisplayName: new("my_service_principal"),
			},
			expected: "my-service-principal",
		},
		{
			name: "uppercase letters should be lowercased",
			user: &scim.User{
				UserName: new("John.Doe@COMPANY.COM"),
			},
			expected: "john-doe",
		},
		{
			name: "mixed case with numbers",
			user: &scim.User{
				UserName: new("User123.Test@example.com"),
			},
			expected: "user123-test",
		},
		{
			name: "unicode characters with normalization",
			user: &scim.User{
				UserName: new("tést.üser@example.com"),
			},
			expected: "test-user",
		},
		{
			name: "complex special characters",
			user: &scim.User{
				UserName: new("user+tag&more@example.com"),
			},
			expected: "user-tag-more",
		},
		{
			name: "service principal with mixed case display name",
			user: &scim.User{
				UserName:    new(`1706906c-c0a2-4c25-9f57-3a7aa3cb8123`),
				DisplayName: new("My_Service_Principal"),
			},
			expected: "my-service-principal",
		},
		{
			name: "email with multiple consecutive special chars",
			user: &scim.User{
				UserName: new("test...user___name@example.com"),
			},
			expected: "test-user-name",
		},
		{
			name: "chinese characters in email",
			user: &scim.User{
				UserName: new("用户.测试@example.com"),
			},
			expected: "-----",
		},
		{
			name: "japanese characters in email",
			user: &scim.User{
				UserName: new("ユーザー.テスト@example.com"),
			},
			expected: "--------",
		},
		{
			name: "emoji in email",
			user: &scim.User{
				UserName: new("user😀.test🚀@example.com"),
			},
			expected: "user-test",
		},
		{
			name: "mixed latin and chinese characters",
			user: &scim.User{
				UserName: new("john.李明@company.com"),
			},
			expected: "john---",
		},
		{
			name: "service principal with emoji display name",
			user: &scim.User{
				UserName:    new("1706906c-c0a2-4c25-9f57-3a7aa3cb8123"),
				DisplayName: new("bot🤖_service"),
			},
			expected: "bot-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetShortUserDomainFriendlyName(tt.user))
		})
	}
}
