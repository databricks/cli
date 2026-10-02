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
		{"environment wrapper", "env NODE_ENV=production pnpm run server", "env NODE_ENV=production npm run server"},
		{"environment wrapper shorthand", `env NODE_ENV=production LABEL="hello world" pnpm server`, `env NODE_ENV=production LABEL="hello world" npm run server`},
		{"environment wrapper options", "env -i -u OLD --unset=OTHER -- NODE_ENV=production pnpm run server", "env -i -u OLD --unset=OTHER -- NODE_ENV=production npm run server"},
		{"environment wrapper in chain", "env pnpm run server && env NODE_ENV=production pnpm run client", "env npm run server && env NODE_ENV=production npm run client"},
		{"environment wrapper script arguments", "env NODE_ENV=production pnpm run server --port 1234", "env NODE_ENV=production npm run server -- --port 1234"},
		{"environment wrapper argument", "env echo pnpm run server", "env echo pnpm run server"},
		{"environment wrapper option value", "env -u pnpm echo pnpm run server", "env -u pnpm echo pnpm run server"},
		{"environment wrapper quoted value", `env LABEL="pnpm run server" echo pnpm run server`, `env LABEL="pnpm run server" echo pnpm run server`},
		{"environment wrapper as argument", "echo env pnpm run server", "echo env pnpm run server"},
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
		{"pnpm", "npm run missing --if-present", "pnpm run --if-present missing"},
		{"pnpm", "npm run task --silent", "pnpm run --silent task"},
		{"pnpm", "npm --silent run task", "pnpm run --silent task"},
		{"pnpm", "npm run --if-present missing", "pnpm run --if-present missing"},
		{"pnpm", "npm --silent run --if-present missing", "pnpm run --silent --if-present missing"},
		{"pnpm", "npm run task first --silent second -- --if-present", "pnpm run --silent task first second --if-present"},
		{"pnpm", "npm run task --if-present --silent -- --silent", "pnpm run --if-present --silent task --silent"},
		{"pnpm", "npm run task --if-present=false --silent=true", "pnpm run --if-present=false --silent=true task"},
		{"pnpm", "npm run task -s first -- --silent", "pnpm run -s task first --silent"},
		{"pnpm", "npm run task --loglevel warn first", "pnpm run --loglevel warn task first"},
		{"pnpm", `npm run task --script-shell "custom shell" first`, `pnpm run --script-shell "custom shell" task first`},
		{"pnpm", `npm run task '--silent' -- "--if-present"`, `pnpm run '--silent' task "--if-present"`},
		{"pnpm", "npm run task --silent && npm run missing --if-present # done", "pnpm run --silent task && pnpm run --if-present missing # done"},
		{"pnpm", "env NODE_ENV=production npm run task --silent -- --silent", "env NODE_ENV=production pnpm run --silent task --silent"},
		{"npm", "pnpm run --if-present missing", "npm run --if-present missing"},
		{"npm", "pnpm --silent run --if-present task --silent", "npm run --silent --if-present task -- --silent"},
		{"npm", "pnpm run task --silent", "npm run task -- --silent"},
		{"npm", "npm run task --silent", "npm run task --silent"},
		{"pnpm", "npm run --silent", "npm run --silent"},
		{"pnpm", "npm run missing --if-present false", "pnpm run --if-present false missing"},
		{"pnpm", "npm run task --if-present true", "pnpm run --if-present true task"},
		{"pnpm", "npm --if-present false run missing", "pnpm run --if-present false missing"},
		{"pnpm", "npm run --if-present false missing", "pnpm run --if-present false missing"},
		{"pnpm", `npm run task --if-present "false" first`, `pnpm run --if-present "false" task first`},
		{"pnpm", `npm run task --if-present f'al'se first`, `pnpm run --if-present f'al'se task first`},
		{"pnpm", `npm run task --if-present f\alse`, `pnpm run --if-present f\alse task`},
		{"pnpm", "npm run task --if-present true -- true false", "pnpm run --if-present true task true false"},
		{"pnpm", "npm run task --if-present=false true", "pnpm run --if-present=false task true"},
		{"pnpm", `npm run task --if-present '"false"'`, `pnpm run --if-present task '"false"'`},
		{"pnpm", `npm run task --if-present "fa\lse"`, `pnpm run --if-present task "fa\lse"`},
		{"pnpm", "npm run task --no-if-present true", "pnpm run --no-if-present true task"},
		{"pnpm", "npm run missing --no-if-present false", "pnpm run --no-if-present false missing"},
		{"pnpm", "npm run task --silent false", "pnpm run --silent task false"},
		{"pnpm", "npm run task -s true", "pnpm run -s task true"},
		{"pnpm", "npm run missing --if-present false || npm run task --if-present true", "pnpm run --if-present false missing || pnpm run --if-present true task"},
		{"npm", "pnpm run --if-present false missing", "npm run --if-present false missing"},
		{"npm", "pnpm run --if-present true task --silent", "npm run --if-present true task -- --silent"},
		{"pnpm", `npm run task '"--label"'`, `pnpm run task '"--label"'`},
		{"pnpm", `npm run task "'--label'"`, `pnpm run task "'--label'"`},
		{"pnpm", `npm run task '""--label""'`, `pnpm run task '""--label""'`},
		{"pnpm", `npm run task \"--label\"`, `pnpm run task \"--label\"`},
		{"pnpm", `npm run task '"--"' --silent`, `pnpm run --silent task '"--"'`},
		{"pnpm", `npm run task ""--silent`, `pnpm run ""--silent task`},
		{"pnpm", `npm run task '--if-'present false`, `pnpm run '--if-'present false task`},
		{"pnpm", `npm run task --script-"shell" "custom shell" first`, `pnpm run --script-"shell" "custom shell" task first`},
		{"pnpm", `npm run task -'-' --if-present false`, "pnpm run task --if-present false"},
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
