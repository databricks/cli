package clijson_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/clijson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func copyShards(t *testing.T) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "checkout[1]")
	require.NoError(t, os.CopyFS(directory, os.DirFS("testdata/shards")))
	return directory
}

func changeIndex(t *testing.T, directory string, change func(*clijson.ShardIndex)) {
	t.Helper()
	path := filepath.Join(directory, "index.json")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	var index clijson.ShardIndex
	require.NoError(t, json.Unmarshal(content, &index))
	change(&index)
	content, err = json.Marshal(index)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, content, 0o644))
}

func replaceShard(t *testing.T, directory, path, content string, rehash bool) {
	t.Helper()
	destination := filepath.Join(directory, filepath.FromSlash(path))
	require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0o755))
	require.NoError(t, os.WriteFile(destination, []byte(content), 0o644))
	if rehash {
		digest := sha256.Sum256([]byte(content))
		changeIndex(t, directory, func(index *clijson.ShardIndex) {
			for _, entries := range [][]clijson.ShardJSON{index.Schemas, index.Services} {
				for position := range entries {
					if path == "schemas/"+entries[position].ID+".json" ||
						path == "commands/"+strings.ReplaceAll(entries[position].ID, ".", "/")+".json" {
						entries[position].SHA256 = hex.EncodeToString(digest[:])
					}
				}
			}
		})
	}
}

func TestReadDir(t *testing.T) {
	directory := copyShards(t)
	doc, err := clijson.Parse(directory)
	require.NoError(t, err)
	require.Len(t, doc.Schemas, 3)
	assert.Equal(t, "An unreferenced schema must not disappear", doc.Schemas["alpha.Unused"].Description)
	assert.Equal(t, []string{"READY", "STOPPED"}, doc.Schemas["zeta.State"].Enum)
	field := doc.Schemas["alpha.Request"].Fields["state"]
	assert.Equal(t, "zeta.State", field.Ref)
	assert.Equal(t, []string{"IMMUTABLE"}, field.Behaviors)
	assert.True(t, field.Deprecated)
	assert.Equal(t, "PUBLIC_PREVIEW", field.LaunchStage)
	require.Len(t, doc.Commands.Services, 2)
	assert.Equal(t, "zeta.Second", doc.Commands.Services[0].ID)
	assert.Equal(t, "alpha.First", doc.Commands.Services[1].ID)
	assert.Equal(t, "Identity", doc.Commands.AccountDocsGroups[0].DisplayName)
	assert.Equal(t, "Compute", doc.Commands.WorkspaceDocsGroups[0].DisplayName)
}

func TestReadDirRejectsInvalidFiles(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		remove  bool
		rehash  bool
	}{
		{name: "missing index", path: "index.json", remove: true},
		{name: "missing schema package", path: "schemas/zeta.json", remove: true},
		{name: "missing service", path: "commands/zeta/Second.json", remove: true},
		{name: "deleted unreferenced schema", path: "schemas/alpha.json", content: `{"alpha.Request":{}}`},
		{name: "checksum mismatch", path: "schemas/zeta.json", content: "{}"},
		{name: "invalid index JSON", path: "index.json", content: "{"},
		{name: "invalid schema JSON", path: "schemas/zeta.json", content: "{", rehash: true},
		{name: "invalid service JSON", path: "commands/zeta/Second.json", content: "{", rehash: true},
		{name: "null schemas", path: "schemas/zeta.json", content: "null", rehash: true},
		{name: "empty schemas", path: "schemas/zeta.json", content: "{}", rehash: true},
		{name: "null schema", path: "schemas/zeta.json", content: `{"zeta.State":null}`, rehash: true},
		{name: "wrong package", path: "schemas/zeta.json", content: `{"alpha.State":{}}`, rehash: true},
		{name: "invalid schema name", path: "schemas/zeta.json", content: `{"zeta.A.B":{}}`, rehash: true},
		{name: "wrong service ID", path: "commands/zeta/Second.json", content: `{"id":"alpha.First"}`, rehash: true},
		{name: "extra schema", path: "schemas/stale.json", content: "{}"},
		{name: "extra nested service", path: "commands/extra/Stale.json", content: "{}"},
		{name: "oversized schema", path: "schemas/zeta.json", content: strings.Repeat(" ", clijson.MaxShardBytes)},
		{name: "oversized index", path: "index.json", content: strings.Repeat(" ", clijson.MaxShardBytes)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := copyShards(t)
			if test.remove {
				require.NoError(t, os.Remove(filepath.Join(directory, test.path)))
			} else {
				replaceShard(t, directory, test.path, test.content, test.rehash)
			}
			_, err := clijson.ReadDir(directory)
			assert.Error(t, err)
		})
	}
	t.Run("missing directory", func(t *testing.T) {
		_, err := clijson.ReadDir(filepath.Join(t.TempDir(), "missing"))
		assert.Error(t, err)
	})
}

func TestReadDirRejectsInvalidIndex(t *testing.T) {
	tests := []struct {
		name   string
		change func(*clijson.ShardIndex)
	}{
		{"unsupported version", func(index *clijson.ShardIndex) { index.Version++ }},
		{"missing version", func(index *clijson.ShardIndex) { index.Version = 0 }},
		{"empty schemas", func(index *clijson.ShardIndex) { index.Schemas = nil }},
		{"empty services", func(index *clijson.ShardIndex) { index.Services = nil }},
		{"missing commands", func(index *clijson.ShardIndex) { index.Commands = nil }},
		{"embedded services", func(index *clijson.ShardIndex) {
			index.Commands.Services = []*clijson.ServiceJSON{{ID: "zeta.Second"}}
		}},
		{"duplicate schema package", func(index *clijson.ShardIndex) { index.Schemas = append(index.Schemas, index.Schemas[0]) }},
		{"duplicate service", func(index *clijson.ShardIndex) { index.Services = append(index.Services, index.Services[0]) }},
		{"invalid schema ID", func(index *clijson.ShardIndex) { index.Schemas[0].ID = "../escape" }},
		{"invalid service ID", func(index *clijson.ShardIndex) { index.Services[0].ID = "zeta.A.B" }},
		{"missing checksum", func(index *clijson.ShardIndex) { index.Schemas[0].SHA256 = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := copyShards(t)
			changeIndex(t, directory, test.change)
			_, err := clijson.ReadDir(directory)
			assert.Error(t, err)
		})
	}
}

func TestReadDirPreservesUnknownFieldsAndNumbers(t *testing.T) {
	directory := copyShards(t)
	replaceShard(t, directory, "schemas/zeta.json", `{"zeta.State":{"future":9007199254740993}}`, true)
	replaceShard(t, directory, "commands/zeta/Second.json", `{"id":"zeta.Second","future":9007199254740993}`, true)
	path := filepath.Join(directory, "index.json")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	var index map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(content, &index))
	index["commands"] = json.RawMessage(`{"future":9007199254740993}`)
	content, err = json.Marshal(index)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, content, 0o644))
	content, err = clijson.ReadDir(directory)
	require.NoError(t, err)
	assert.Equal(t, 3, strings.Count(string(content), `"future":9007199254740993`))
}
