package annotation_test

import (
	"testing"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/stretchr/testify/assert"
)

func TestPreviewTag(t *testing.T) {
	tests := []struct {
		launchStage string
		want        string
	}{
		{"PUBLIC_PREVIEW", "[Public Preview]"},
		{"PUBLIC_BETA", "[Beta]"},
		{"PRIVATE_PREVIEW", "[Private Preview]"},
		{"GA", ""},
		{"", ""},
		{"SOMETHING_ELSE", ""},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, annotation.PreviewTag(tc.launchStage))
	}
}

func TestPreviewTagShort(t *testing.T) {
	tests := []struct {
		launchStage string
		want        string
	}{
		{"PUBLIC_PREVIEW", "[PuPr]"},
		{"PUBLIC_BETA", "[Beta]"},
		{"PRIVATE_PREVIEW", "[PrPr]"},
		{"GA", ""},
		{"", ""},
		{"SOMETHING_ELSE", ""},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, annotation.PreviewTagShort(tc.launchStage))
	}
}
