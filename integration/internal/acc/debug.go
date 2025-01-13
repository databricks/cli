package acc

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/internal/testutil"
)

// Detects if test is run from "debug test" feature in VS Code.
func IsInDebug() bool {
	ex, _ := os.Executable()
	return strings.HasPrefix(path.Base(ex), "__debug_bin")
}

// Loads debug environment from ~/.databricks/debug-env.json.
func loadDebugEnvIfRunFromIDE(t testutil.TestingT, key string) {
	if !IsInDebug() {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot find user home: %s", err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".databricks/debug-env.json"))
	if err != nil {
		t.Fatalf("cannot load ~/.databricks/debug-env.json: %s", err)
	}
	var conf map[string]map[string]string
	err = json.Unmarshal(raw, &conf)
	if err != nil {
		t.Fatalf("cannot parse ~/.databricks/debug-env.json: %s", err)
	}
	vars, ok := conf[key]
	if !ok {
		t.Fatalf("~/.databricks/debug-env.json#%s not configured", key)
	}
	for k, v := range vars {
		os.Setenv(k, v)
	}
}
