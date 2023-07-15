package mutator

import "testing"

func TestPopulateCurrentUser(t *testing.T) {
	// We need to implement workspace client mocking to implement this test.
}

func TestGetShortUserName(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "test alphanumeric characters",
			email:    "test.user@example.com",
			expected: "test_user",
		},
		{
			name:     "test unicode characters",
			email:    "tést.üser@example.com",
			expected: "tést_üser",
		},
		{
			name:     "test special characters",
			email:    "test$.user@example.com",
			expected: "test__user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getShortUserName(tt.email)
			if result != tt.expected {
				t.Errorf("getShortUserName(%q) = %q; expected %q", tt.email, result, tt.expected)
			}
		})
	}
}
