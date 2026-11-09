package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadAppKitInitConfigRejectsInvalidJSON(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     string
	}{
		{"malformed", `{"project_name":`, "unexpected EOF"},
		{"unknown field", `{"project_name":"app","extra":true}`, `unknown field "extra"`},
		{"trailing object", `{"project_name":"app"} {"project_name":"other"}`, "must contain one JSON object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.json")
			require.NoError(t, os.WriteFile(path, []byte(tt.contents), 0o600))
			_, err := readAppKitInitConfig(path)
			require.ErrorContains(t, err, tt.want)
		})
	}
}
