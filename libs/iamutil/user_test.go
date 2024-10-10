package iamutil

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

func TestGetShortUserName(t *testing.T) {
	tests := []struct {
		name     string
		user     *iam.User
		expected string
	}{
		{
			user: &iam.User{
				UserName: "test.user.1234@example.com",
			},
			expected: "test_user_1234",
		},
		{
			user: &iam.User{
				UserName: "tést.üser@example.com",
			},
			expected: "tést_üser",
		},
		{
			user: &iam.User{
				UserName: "test$.user@example.com",
			},
			expected: "test_user",
		},
		{
			user: &iam.User{
				UserName: `jöhn.dœ@domain.com`, // Using non-ASCII characters.
			},
			expected: "jöhn_dœ",
		},
		{
			user: &iam.User{
				UserName: `first+tag@email.com`, // The plus (+) sign is used for "sub-addressing" in some email services.
			},
			expected: "first_tag",
		},
		{
			user: &iam.User{
				UserName: `email@sub.domain.com`, // Using a sub-domain.
			},
			expected: "email",
		},
		{
			user: &iam.User{
				UserName: `"_quoted"@domain.com`, // Quoted strings can be part of the local-part.
			},
			expected: "quoted",
		},
		{
			user: &iam.User{
				UserName: `name-o'mally@website.org`, // Single quote in the local-part.
			},
			expected: "name_o_mally",
		},
		{
			user: &iam.User{
				UserName: `user%domain@external.com`, // Percent sign can be used for email routing in legacy systems.
			},
			expected: "user_domain",
		},
		{
			user: &iam.User{
				UserName: `long.name.with.dots@domain.net`, // Multiple dots in the local-part.
			},
			expected: "long_name_with_dots",
		},
		{
			user: &iam.User{
				UserName: `me&you@together.com`, // Using an ampersand (&) in the local-part.
			},
			expected: "me_you",
		},
		{
			user: &iam.User{
				UserName: `user!def!xyz@domain.org`, // The exclamation mark can be valid in some legacy systems.
			},
			expected: "user_def_xyz",
		},
		{
			user: &iam.User{
				UserName: `admin@ιντερνετ.com`, // Domain in non-ASCII characters (IDN or Internationalized Domain Name).
			},
			expected: "admin",
		},
		{
			user: &iam.User{
				UserName:    `1706906c-c0a2-4c25-9f57-3a7aa3cb8123`,
				DisplayName: "my-service-principal",
			},
			expected: "my_service_principal",
		},
		{
			user: &iam.User{
				UserName: `1706906c-c0a2-4c25-9f57-3a7aa3cb8123`,
				// This service princpal has DisplayName (it's an optional property)
			},
			expected: "1706906c_c0a2_4c25_9f57_3a7aa3cb8123",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, GetShortUserName(tt.user))
	}
}
