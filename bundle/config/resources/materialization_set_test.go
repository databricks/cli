package resources

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMaterializationSetReportsNoWorkspaceURL pins the signal that keeps `bundle summary` from
// printing a URL line for a set. BaseResource.GetURL returns ("", true), which the summary reads as
// a supported type awaiting deployment and renders as "(not deployed)".
func TestMaterializationSetReportsNoWorkspaceURL(t *testing.T) {
	set := &MaterializationSet{ID: "0123456789abcdef0123456789abcdef"}
	set.URL = "https://example.cloud.databricks.com/explore/data/features"

	rawURL, supportsURL := set.GetURL()
	assert.False(t, supportsURL)
	assert.Empty(t, rawURL)

	set.InitializeURL(url.URL{Scheme: "https", Host: "example.cloud.databricks.com"})
	rawURL, supportsURL = set.GetURL()
	assert.False(t, supportsURL, "InitializeURL must not make the type look URL-bearing")
	assert.Empty(t, rawURL)
}
