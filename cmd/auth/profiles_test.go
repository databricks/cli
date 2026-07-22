package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileMetadataIsEmpty(t *testing.T) {
	cases := []struct {
		name  string
		meta  profileMetadata
		empty bool
	}{
		{"no host or account", profileMetadata{Name: "p"}, true},
		{"host set", profileMetadata{Name: "p", Host: "https://x.cloud.databricks.test"}, false},
		{"account set", profileMetadata{Name: "p", AccountID: "acct"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.empty, tc.meta.IsEmpty())
		})
	}
}

func TestCloudFromHost(t *testing.T) {
	cases := []struct {
		host  string
		cloud string
	}{
		{"https://abc.cloud.databricks.com", "aws"},
		{"https://adb-123.4.azuredatabricks.net", "azure"},
		{"https://123.4.gcp.databricks.com", "gcp"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			assert.Equal(t, tc.cloud, cloudFromHost(tc.host))
		})
	}
}
