package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dynMerge merges b into a with the dynamic configuration, like includes do.
func dynMerge(t *testing.T, fa, fb string, a, b []byte) (string, bool) {
	ra, diags := config.LoadDynFromBytes(fa, a)
	if diags.HasError() {
		return "", false
	}
	rb, diags := config.LoadDynFromBytes(fb, b)
	if diags.HasError() {
		return "", false
	}
	if err := ra.Merge(rb); err != nil {
		return "merge error: " + err.Error(), true
	}
	if _, err := ra.FinishLoading(); err != nil {
		return "", false
	}
	out, _ := json.Marshal(ra)
	return string(out), true
}

func typedMerge(t *testing.T, fa, fb string, a, b []byte) string {
	var ra, rb config.Root
	sva, _, err := structvar.DecodeYAML(fa, strings.NewReader(string(a)), &ra)
	if err != nil {
		return "decode error: " + err.Error()
	}
	svb, _, err := structvar.DecodeYAML(fb, strings.NewReader(string(b)), &rb)
	if err != nil {
		return "decode error: " + err.Error()
	}
	if err := sva.Merge(nil, svb.View()); err != nil {
		return "merge error: " + err.Error()
	}
	out, _ := json.Marshal(&ra)
	return string(out)
}

// TestMergeParityWithDyn merges every ordered pair of the YAML files of a directory of
// the acceptance tests (and of its subdirectories) once with the dynamic configuration,
// like includes do, and once with structvar.Merge on the typed configuration, and
// requires the same result. The only difference is the shorthand of a target's variable
// (`foo: value` for `foo: {default: value}`), which loading rewrites before merging.
func TestMergeParityWithDyn(t *testing.T) {
	root := "../../acceptance/bundle"
	same, diff, skipped, sameButVars := 0, 0, 0, 0
	byDir := map[string][]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && (strings.HasSuffix(p, ".yml") || strings.HasSuffix(p, ".yaml")) {
			byDir[filepath.Dir(p)] = append(byDir[filepath.Dir(p)], p)
		}
		return nil
	})
	require.NoError(t, err)
	// Pairs of the files of a directory (and its subdirectories one level down).
	var groups [][]string
	for dir, files := range byDir {
		g := append([]string(nil), files...)
		for d2, f2 := range byDir {
			if filepath.Dir(d2) == dir {
				g = append(g, f2...)
			}
		}
		groups = append(groups, g)
	}
	for _, files := range groups {
		for _, fa := range files {
			a, _ := os.ReadFile(fa)
			for _, fb := range files {
				b, _ := os.ReadFile(fb)
				want, ok := dynMerge(t, fa, fb, a, b)
				if !ok {
					skipped++
					continue
				}
				got := typedMerge(t, fa, fb, a, b)
				if got == want {
					same++
					continue
				}
				if withoutTargetVariables(t, got) == withoutTargetVariables(t, want) {
					sameButVars++
					continue
				}
				diff++
				if diff <= 12 {
					t.Logf("DIFF %s + %s\n dyn:   %.400s\n typed: %.400s", fa, fb, want, got)
				}
			}
		}
	}
	assert.Zero(t, diff)
	t.Logf("same=%d only-target-variables-differ=%d diff=%d skipped=%d", same, sameButVars, diff, skipped)
}

// withoutTargetVariables drops the variables of the targets (the shorthand rewrite).
func withoutTargetVariables(t *testing.T, s string) string {
	var m map[string]any
	if json.Unmarshal([]byte(s), &m) != nil {
		return s
	}
	if ts, ok := m["targets"].(map[string]any); ok {
		for _, tv := range ts {
			if tm, ok := tv.(map[string]any); ok {
				delete(tm, "variables")
			}
		}
	}
	out, _ := json.Marshal(m)
	return string(out)
}
