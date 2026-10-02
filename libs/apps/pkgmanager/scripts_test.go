package pkgmanager_test

import (
	"testing"

	"github.com/databricks/cli/libs/apps/pkgmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteScriptCommands(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   string
	}{
		{"compound build", "pnpm run server && pnpm run client", "npm run server && npm run client"},
		{"compound predev", "pnpm run sync && pnpm run typegen", "npm run sync && npm run typegen"},
		{"no spaces around separator", "pnpm run server&&pnpm run client", "npm run server&&npm run client"},
		{"conditional", "pnpm run server || pnpm run client", "npm run server || npm run client"},
		{"multiple statements", "pnpm run server; pnpm run client", "npm run server; npm run client"},
		{"pipeline", "pnpm run server | pnpm run client", "npm run server | npm run client"},
		{"line break", "pnpm run server\npnpm run client", "npm run server\nnpm run client"},
		{"after another tool", "node --version && pnpm run server", "node --version && npm run server"},
		{"leading whitespace", "  pnpm run server", "  npm run server"},
		{"tabs", "pnpm\trun\tserver", "npm\trun\tserver"},
		{"shorthand", "pnpm server && pnpm client", "npm run server && npm run client"},
		{"native clean install", "npm ci", "npm ci"},
		{"native exec", "npm exec tsc", "npm exec tsc"},
		{"native lifecycle name", "npm install", "npm install"},
		{"bare manager", "pnpm", "pnpm"},
		{"list scripts", "pnpm run", "pnpm run"},
		{"list scripts before comment", "pnpm run # list scripts", "pnpm run # list scripts"},
		{"quoted argument", `echo "pnpm run server && pnpm run client" && pnpm run server`, `echo "pnpm run server && pnpm run client" && npm run server`},
		{"single quoted argument", `node -e 'console.log("pnpm run server && pnpm run client")' && pnpm run server`, `node -e 'console.log("pnpm run server && pnpm run client")' && npm run server`},
		{"command argument", "echo pnpm run server", "echo pnpm run server"},
		{"script arguments", `pnpm run server -- --label "pnpm run client"`, `npm run server -- -- --label "pnpm run client"`},
		{"environment assignment", "NODE_ENV=production pnpm run server", "NODE_ENV=production npm run server"},
		{"multiple assignments", `NODE_ENV=production LABEL="hello world" pnpm run server`, `NODE_ENV=production LABEL="hello world" npm run server`},
		{"assignment in chain", "pnpm run server && NODE_ENV=production pnpm run client", "npm run server && NODE_ENV=production npm run client"},
		{"assignment argument", "echo NODE_ENV=production pnpm run server", "echo NODE_ENV=production pnpm run server"},
		{"escaped separator", `echo pnpm\;pnpm run server && pnpm run client`, `echo pnpm\;pnpm run server && npm run client`},
		{"Windows escaped separator", `echo pnpm^&pnpm run server && pnpm run client`, `echo pnpm^&pnpm run server && npm run client`},
	}
	m, err := pkgmanager.Resolve("npm")
	require.NoError(t, err)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := map[string]any{"scripts": map[string]any{
				"check": tt.script, "server": "node server.js", "client": "node client.js",
				"install": "node setup.js",
			}}
			pkgmanager.Rewrite(pkg, m)
			assert.Equal(t, tt.want, pkg["scripts"].(map[string]any)["check"])
		})
	}
}

func TestRewriteScriptArguments(t *testing.T) {
	tests := []struct {
		manager string
		script  string
		want    string
	}{
		{"npm", "pnpm run task --port 1234", "npm run task -- --port 1234"},
		{"npm", `pnpm run task --label "hello world"`, `npm run task -- --label "hello world"`},
		{"npm", "pnpm task --port 1234", "npm run task -- --port 1234"},
		{"npm", "pnpm run task first -- --port 1234", "npm run task -- first -- --port 1234"},
		{"npm", "pnpm run task # comment", "npm run task # comment"},
		{"npm", "pnpm run task --port 1234 && pnpm run task --port 5678", "npm run task -- --port 1234 && npm run task -- --port 5678"},
		{"pnpm", "npm run task -- --port 1234", "pnpm run task --port 1234"},
		{"pnpm", `npm run task -- --label "hello world"`, `pnpm run task --label "hello world"`},
		{"pnpm", "npm run task first -- --port 1234", "pnpm run task first --port 1234"},
		{"pnpm", "npm run task --", "pnpm run task"},
		{"pnpm", "npm run task -- -- --port 1234", "pnpm run task -- --port 1234"},
		{"pnpm", `npm run task "--" --port 1234`, "pnpm run task --port 1234"},
		{"pnpm", "npm run task -- --port 1234 && npm run task -- --port 5678", "pnpm run task --port 1234 && pnpm run task --port 5678"},
		{"npm", "npm run task -- --port 1234", "npm run task -- --port 1234"},
		{"pnpm", "pnpm run task --port 1234", "pnpm run task --port 1234"},
	}
	for _, tt := range tests {
		t.Run(tt.manager+"/"+tt.script, func(t *testing.T) {
			m, err := pkgmanager.Resolve(tt.manager)
			require.NoError(t, err)
			pkg := map[string]any{"scripts": map[string]any{"check": tt.script, "task": "node task.js"}}
			pkgmanager.Rewrite(pkg, m)
			assert.Equal(t, tt.want, pkg["scripts"].(map[string]any)["check"])
		})
	}
}
