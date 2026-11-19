package testproxy

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/httptransport"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestProxyUsesHealthCheckedTransport(t *testing.T) {
	server := testserver.New(t)
	upstream := &config.Config{Host: server.URL, Token: "token"}
	New(t, upstream)
	assert.Same(t, httptransport.HealthChecked(false), upstream.HTTPTransport)
}

func TestProxyKeepsCallerTransport(t *testing.T) {
	server := testserver.New(t)
	transport := &http.Transport{}
	upstream := &config.Config{Host: server.URL, Token: "token", HTTPTransport: transport}
	New(t, upstream)
	assert.Same(t, transport, upstream.HTTPTransport)
}
