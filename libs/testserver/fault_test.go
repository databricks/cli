package testserver_test

import (
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFaultRulesNoMatch(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "GET /foo", 504, "body", 0, 0, 1)

	assert.Nil(t, fr.Check("POST", "/foo", "tok"))
	assert.Nil(t, fr.Check("GET", "/bar", "tok"))
	assert.Nil(t, fr.Check("GET", "/foo", "other"))
}

func TestFaultRulesExactMatch(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "PUT /api/2.0/jobs/123", 504, "body", 0, 0, 1)

	rule := fr.Check("PUT", "/api/2.0/jobs/123", "tok")
	require.NotNil(t, rule)
	assert.Equal(t, 504, rule.StatusCode)
	assert.Equal(t, "body", rule.Body)
}

func TestFaultRulesDelay(t *testing.T) {
	fr := testserver.NewFaultRules()
	// A delay-only rule (status code 0) carries the delay but no error status, so
	// the caller sleeps and then falls through to the real handler.
	fr.Set("tok", "POST /api/2.0/slow", 0, "", 250, 0, 1)

	rule := fr.Check("POST", "/api/2.0/slow", "tok")
	require.NotNil(t, rule)
	assert.Equal(t, 0, rule.StatusCode)
	assert.Equal(t, 250, rule.DelayMs)
}

func TestFaultRulesWildcardMatch(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "PUT /api/2.0/permissions/pipelines/*", 504, "body", 0, 0, 2)

	assert.NotNil(t, fr.Check("PUT", "/api/2.0/permissions/pipelines/abc", "tok"))
	assert.NotNil(t, fr.Check("PUT", "/api/2.0/permissions/pipelines/xyz", "tok"))
	assert.Nil(t, fr.Check("PUT", "/api/2.0/permissions/pipelines/xyz", "tok")) // exhausted
}

func TestFaultRulesOffset(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "GET /foo", 504, "body", 0, 2, 1)

	assert.Nil(t, fr.Check("GET", "/foo", "tok"))    // offset 2→1
	assert.Nil(t, fr.Check("GET", "/foo", "tok"))    // offset 1→0
	assert.NotNil(t, fr.Check("GET", "/foo", "tok")) // fires
	assert.Nil(t, fr.Check("GET", "/foo", "tok"))    // exhausted
}

func TestFaultRulesTimes(t *testing.T) {
	fr := testserver.NewFaultRules()
	fr.Set("tok", "GET /foo", 504, "body", 0, 0, 3)

	for range 3 {
		assert.NotNil(t, fr.Check("GET", "/foo", "tok"))
	}
	assert.Nil(t, fr.Check("GET", "/foo", "tok")) // exhausted
}
