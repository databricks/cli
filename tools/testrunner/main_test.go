package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigRuleMatches(t *testing.T) {
	tests := []struct {
		input       string
		packageName string
		testcase    string
		match       bool
	}{
		// Exact matches
		{"bundle TestDeploy", "bundle", "TestDeploy", true},
		{"bundle TestDeploy", "libs", "TestDeploy", false},
		{"bundle TestDeploy", "bundle", "TestSomethingElse", false},

		// Package prefix matches
		{"libs/ TestSomething", "libs/auth", "TestSomething", true},
		{"libs/ TestSomething", "libs", "TestSomething", true},
		{"libs/ TestSomething", "libsother", "TestSomething", false},

		// Test prefix matches
		{"bundle TestAccept/", "bundle", "TestAcceptDeploy", false},
		{"bundle TestAccept/", "bundle", "TestAccept", true},
		{"bundle TestAccept/", "bundle", "TestAccept/Deploy", true},

		// Wildcard matches
		{"* *", "any/package", "AnyTest", true},
		{"* TestAccept/", "any/package", "TestAcceptDeploy", false},
		{"* TestAccept/", "any/package", "TestAccept/Deploy", true},
		{"libs/ *", "libs/auth", "AnyTest", true},

		// Path prefix edge cases
		{"TestAccept/ TestAccept/", "TestAccept", "TestAccept", true},
		{"TestAccept/ TestAccept/", "TestAccept/bundle", "TestAccept/deploy", true},
		{"TestAccept/ TestAccept/", "TestAcceptSomething", "TestAcceptSomething", false},

		// Empty values cases
		{"* TestDeploy", "", "TestDeploy", true},
		{"bundle *", "bundle", "", true},

		// Subtest failure results in parent test failure as well. So we allow strict prefixes to fail as well
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic", "acceptance", "TestAccept", true},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic", "acceptance", "TestAnother", false},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic", "acceptance", "TestAccept/bundle/templates/default-python/combinations/classic/x", false},

		// pattern version
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic/", "acceptance", "TestAccept", true},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic/", "acceptance", "TestAnother", false},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic/", "acceptance", "TestAccept/bundle/templates/default-python/combinations/classic/x", true},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic/", "acceptance", "TestAccept/bundle/templates/default-python/combinations/classic", true},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic/", "acceptance", "TestAccept/bundle/templates/default-python/combinations", true},
	}

	for _, tt := range tests {
		t.Run(tt.input+"_"+tt.packageName+"_"+tt.testcase, func(t *testing.T) {
			rule, err := parseConfigRule(tt.input, tt.input)
			assert.NoError(t, err)
			result := rule.matches(tt.packageName, tt.testcase)
			assert.Equal(t, tt.match, result)
		})
	}
}
