package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNamesTests(t *testing.T) {
	for _, msg := range []string{
		"run tests\n\naws: TestAccept/bundle/resources/jobs\n",
		"cloud: TestAccept/a",
		"gcp:TestAccept/a",
		"  azure: TestAccept/a   \n",
		"aws: TestAccept",
		"aws: TestFoo TestAccept/a",
	} {
		assert.True(t, NamesTests(msg), msg)
	}

	for _, msg := range []string{
		"",
		"Fix a bug",
		"dbr: TestAccept/a",
		"AWS: TestAccept/a",
		"see aws: TestAccept/a",
		"aws: TestAcceptance/a",
		"aws: TestFoo",
		"aws: fix flaky TestAccept retries",
		"aws:",
	} {
		assert.False(t, NamesTests(msg), msg)
	}
}
