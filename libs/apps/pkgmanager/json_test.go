package pkgmanager_test

import (
	"testing"

	"github.com/databricks/cli/libs/apps/pkgmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteJSON(t *testing.T) {
	for _, tt := range []struct {
		name    string
		manager string
		input   string
		want    string
	}{
		{
			name:    "preserve order, numbers, nested formatting, and literal operators",
			manager: "npm",
			input:   "{\n  \"version\": \"1.0\",\n  \"name\": \"old\",\n  \"scripts\": {\"z\": \"pnpm run a && echo '<&>'\", \"a\": \"echo done\"},\n  \"custom\": {\"z\": 9007199254740993, \"a\": [1,  2]},\n  \"packageManager\": \"npm@10.8.3\"\n}\n",
			want:    "{\n  \"version\": \"1.0\",\n  \"name\": \"new\",\n  \"scripts\": {\"z\": \"npm run a && echo '<&>'\", \"a\": \"echo done\"},\n  \"custom\": {\"z\": 9007199254740993, \"a\": [1,  2]},\n  \"packageManager\": \"npm@10.8.3\"\n}\n",
		},
		{
			name:    "preserve pnpm pin and original escapes",
			manager: "pnpm",
			input:   `{"packageManager":"pnpm@10.17.1+sha224.abc","name":"new","scripts":{"dev":"echo \u0026"}}`,
			want:    `{"packageManager":"pnpm@10.17.1+sha224.abc","name":"new","scripts":{"dev":"echo \u0026"}}`,
		},
		{
			name:    "insert pin with tabs and CRLF",
			manager: "pnpm",
			input:   "{\r\n\t\"name\": \"old\",\r\n\t\"version\": \"1\"\r\n}\r\n",
			want:    "{\r\n\t\"name\": \"new\",\r\n\t\"version\": \"1\",\r\n\t\"packageManager\": \"pnpm@11.0.8\"\r\n}\r\n",
		},
		{
			name:    "remove conflicting pin",
			manager: "npm",
			input:   "{\n  \"packageManager\": \"pnpm@10.17.1\",\n  \"name\": \"old\"\n}\n",
			want:    "{\n  \"name\": \"new\"\n}\n",
		},
		{
			name:    "remove final pin",
			manager: "npm",
			input:   `{"name":"old","packageManager":"pnpm@11.0.8"}`,
			want:    `{"name":"new"}`,
		},
		{
			name:    "empty object",
			manager: "pnpm",
			input:   `{}`,
			want:    `{"name":"new","packageManager":"pnpm@11.0.8"}`,
		},
		{
			name:    "empty multiline object",
			manager: "pnpm",
			input:   "{\n}\n",
			want:    "{\n  \"name\": \"new\",\n  \"packageManager\": \"pnpm@11.0.8\"\n}\n",
		},
		{
			name:    "script names with pointer characters and non-string values",
			manager: "npm",
			input:   `{"name":"old","scripts":{"a/b~c":"pnpm run dev && pnpm run dev","dev":"node app.js","other":{"nested":1}}}`,
			want:    `{"name":"new","scripts":{"a/b~c":"npm run dev && npm run dev","dev":"node app.js","other":{"nested":1}}}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, err := pkgmanager.Resolve(tt.manager)
			require.NoError(t, err)
			got, err := pkgmanager.RewriteJSON([]byte(tt.input), "new", m)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
			again, err := pkgmanager.RewriteJSON(got, "new", m)
			require.NoError(t, err)
			assert.Equal(t, got, again)
		})
	}
}

func TestRewriteJSONInvalid(t *testing.T) {
	for _, input := range []string{`{`, `[]`, `null`, `{"name":"old",}`, `{"name":/* comment */"old"}`} {
		t.Run(input, func(t *testing.T) {
			_, err := pkgmanager.RewriteJSON([]byte(input), "new", pkgmanager.Default())
			require.Error(t, err)
		})
	}
}
