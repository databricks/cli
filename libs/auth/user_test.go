package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetShortUserName(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			email:    "test.user.1234@example.com",
			expected: "test_user_1234",
		},
		{
			email:    "tést.üser@example.com",
			expected: "tést_üser",
		},
		{
			email:    "test$.user@example.com",
			expected: "test__user",
		},
		{
			email:    `jöhn.dœ@domain.com`, // Using non-ASCII characters.
			expected: "jöhn_dœ",
		},
		{
			email:    `first+tag@email.com`, // The plus (+) sign is used for "sub-addressing" in some email services.
			expected: "first_tag",
		},
		{
			email:    `email@sub.domain.com`, // Using a sub-domain.
			expected: "email",
		},
		{
			email:    `"_quoted"@domain.com`, // Quoted strings can be part of the local-part.
			expected: "__quoted_",
		},
		{
			email:    `name-o'mally@website.org`, // Single quote in the local-part.
			expected: "name_o_mally",
		},
		{
			email:    `user%domain@external.com`, // Percent sign can be used for email routing in legacy systems.
			expected: "user_domain",
		},
		{
			email:    `long.name.with.dots@domain.net`, // Multiple dots in the local-part.
			expected: "long_name_with_dots",
		},
		{
			email:    `me&you@together.com`, // Using an ampersand (&) in the local-part.
			expected: "me_you",
		},
		{
			email:    `user!def!xyz@domain.org`, // The exclamation mark can be valid in some legacy systems.
			expected: "user_def_xyz",
		},
		{
			email:    `admin@ιντερνετ.com`, // Domain in non-ASCII characters (IDN or Internationalized Domain Name).
			expected: "admin",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, GetShortUserName(tt.email))
	}
}
