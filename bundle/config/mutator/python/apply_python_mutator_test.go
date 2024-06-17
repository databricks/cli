package python

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/dyn"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/cli/libs/process"
)

func TestApplyPythonMutator_Name_preinit(t *testing.T) {
	mutator := ApplyPythonMutator(ApplyPythonMutatorPhasePreInit)

	assert.Equal(t, "ApplyPythonMutator(preinit)", mutator.Name())
}

func TestApplyPythonMutator_Name_init(t *testing.T) {
	mutator := ApplyPythonMutator(ApplyPythonMutatorPhaseInit)

	assert.Equal(t, "ApplyPythonMutator(init)", mutator.Name())
}

func TestApplyPythonMutator_preinit(t *testing.T) {
	withFakeVEnv(t, ".venv")

	b := loadYaml("databricks.yml", `
      experimental:
        pydabs:
          enable: true
        venv:
          path: .venv
      resources:
        jobs:
          job0:
            name: job_0`)

	ctx := withProcessStub(
		[]string{
			interpreterPath(".venv"),
			"-m",
			"databricks.bundles.build",
			"--phase",
			"preinit",
		},
		`{
			"experimental": {
				"pydabs": { "enable": true },
				"venv": { "path": ".venv" }
			},
			"resources": {
				"jobs": {
					"job0": {
						name: "job_0"
					},
					"job1": {
						name: "job_1"
					},
				}
			}
		}`)

	mutator := ApplyPythonMutator(ApplyPythonMutatorPhasePreInit)
	diag := bundle.Apply(ctx, b, mutator)

	assert.NoError(t, diag.Error())

	assert.ElementsMatch(t, []string{"job0", "job1"}, keys(b.Config.Resources.Jobs))

	if job0, ok := b.Config.Resources.Jobs["job0"]; ok {
		assert.Equal(t, "job_0", job0.Name)
	}

	if job1, ok := b.Config.Resources.Jobs["job1"]; ok {
		assert.Equal(t, "job_1", job1.Name)
	}
}

func TestApplyPythonMutator_preinit_disallowed(t *testing.T) {
	withFakeVEnv(t, ".venv")

	b := loadYaml("databricks.yml", `
      experimental:
        pydabs:
          enable: true
        venv:
          path: .venv
      resources:
        jobs:
          job0:
            name: job_0`)

	ctx := withProcessStub(
		[]string{
			".venv/bin/python3",
			"-m",
			"databricks.bundles.build",
			"--phase",
			"preinit",
		},
		`{
			"experimental": {
				"pydabs": { "enable": true },
				"venv": { "path": ".venv" }
			},
			"resources": {
				"jobs": {
					"job0": {
						name: "job_0",
						description: "job description"
					}
				}
			}
		}`)

	mutator := ApplyPythonMutator(ApplyPythonMutatorPhasePreInit)
	diag := bundle.Apply(ctx, b, mutator)

	assert.EqualError(t, diag.Error(), "unexpected change at \"resources.jobs.job0.description\" (insert)")
}

func TestApplyPythonMutator_init(t *testing.T) {
	withFakeVEnv(t, ".venv")

	b := loadYaml("databricks.yml", `
      experimental:
        pydabs:
          enable: true
        venv:
          path: .venv
      resources:
        jobs:
          job0:
            name: job_0`)

	ctx := withProcessStub(
		[]string{
			interpreterPath(".venv"),
			"-m",
			"databricks.bundles.build",
			"--phase",
			"init",
		},
		`{
			"experimental": {
				"pydabs": { "enable": true },
				"venv": { "path": ".venv" }
			},
			"resources": {
				"jobs": {
					"job0": {
						name: "job_0",
						description: "my job"
					}
				}
			}
		}`)

	mutator := ApplyPythonMutator(ApplyPythonMutatorPhaseInit)
	diag := bundle.Apply(ctx, b, mutator)

	assert.NoError(t, diag.Error())

	assert.ElementsMatch(t, []string{"job0"}, keys(b.Config.Resources.Jobs))
	assert.Equal(t, "job_0", b.Config.Resources.Jobs["job0"].Name)
	assert.Equal(t, "my job", b.Config.Resources.Jobs["job0"].Description)
}

func TestApplyPythonMutator_badOutput(t *testing.T) {
	withFakeVEnv(t, ".venv")

	b := loadYaml("databricks.yml", `
      experimental:
        pydabs:
          enable: true
        venv:
          path: .venv
      resources:
        jobs:
          job0:
            name: job_0`)

	ctx := withProcessStub(
		[]string{
			interpreterPath(".venv"),
			"-m",
			"databricks.bundles.build",
			"--phase",
			"preinit",
		},
		`{
			"resources": {
				"jobs": {
					"job0": {
						unknown_property: "my job"
					}
				}
			}
		}`)

	mutator := ApplyPythonMutator(ApplyPythonMutatorPhasePreInit)
	diag := bundle.Apply(ctx, b, mutator)

	assert.EqualError(t, diag.Error(), "failed to normalize Python mutator output: unknown field: unknown_property")
}

func TestApplyPythonMutator_disabled(t *testing.T) {
	b := loadYaml("databricks.yml", ``)

	ctx := context.Background()
	mutator := ApplyPythonMutator(ApplyPythonMutatorPhasePreInit)
	diag := bundle.Apply(ctx, b, mutator)

	assert.NoError(t, diag.Error())
}

func TestApplyPythonMutator_venvRequired(t *testing.T) {
	b := loadYaml("databricks.yml", `
      experimental:
        pydabs:
          enable: true`)

	ctx := context.Background()
	mutator := ApplyPythonMutator(ApplyPythonMutatorPhasePreInit)
	diag := bundle.Apply(ctx, b, mutator)

	assert.Error(t, diag.Error(), "\"experimental.enable_pydabs\" is enabled, but \"experimental.venv.path\" is not set")
}

func TestApplyPythonMutator_venvNotFound(t *testing.T) {
	expectedError := fmt.Sprintf("can't find %q, check if venv is created", interpreterPath("bad_path"))

	b := loadYaml("databricks.yml", `
      experimental:
        pydabs:
          enable: true
        venv:
          path: bad_path`)

	mutator := ApplyPythonMutator(ApplyPythonMutatorPhaseInit)
	diag := bundle.Apply(context.Background(), b, mutator)

	assert.EqualError(t, diag.Error(), expectedError)
}

type createOverrideVisitorTestCase struct {
	name        string
	updatePath  dyn.Path
	deletePath  dyn.Path
	insertPath  dyn.Path
	phase       phase
	updateError error
	deleteError error
	insertError error
}

func TestCreateOverrideVisitor(t *testing.T) {
	left := dyn.NewValue(42, dyn.Location{})
	right := dyn.NewValue(1337, dyn.Location{})

	testCases := []createOverrideVisitorTestCase{
		{
			name:        "preinit: can't change an existing job",
			phase:       ApplyPythonMutatorPhasePreInit,
			updatePath:  dyn.MustPathFromString("resources.jobs.job0.name"),
			deletePath:  dyn.MustPathFromString("resources.jobs.job0.name"),
			insertPath:  dyn.MustPathFromString("resources.jobs.job0.name"),
			deleteError: fmt.Errorf("unexpected change at \"resources.jobs.job0.name\" (delete)"),
			insertError: fmt.Errorf("unexpected change at \"resources.jobs.job0.name\" (insert)"),
			updateError: fmt.Errorf("unexpected change at \"resources.jobs.job0.name\" (update)"),
		},
		{
			name:        "preinit: can't delete an existing job",
			phase:       ApplyPythonMutatorPhasePreInit,
			deletePath:  dyn.MustPathFromString("resources.jobs.job0"),
			deleteError: fmt.Errorf("unexpected change at \"resources.jobs.job0\" (delete)"),
		},
		{
			name:        "preinit: can insert a job",
			phase:       ApplyPythonMutatorPhasePreInit,
			insertPath:  dyn.MustPathFromString("resources.jobs.job0"),
			insertError: nil,
		},
		{
			name:        "preinit: can't change include",
			phase:       ApplyPythonMutatorPhasePreInit,
			deletePath:  dyn.MustPathFromString("include[0]"),
			insertPath:  dyn.MustPathFromString("include[0]"),
			updatePath:  dyn.MustPathFromString("include[0]"),
			deleteError: fmt.Errorf("unexpected change at \"include[0]\" (delete)"),
			insertError: fmt.Errorf("unexpected change at \"include[0]\" (insert)"),
			updateError: fmt.Errorf("unexpected change at \"include[0]\" (update)"),
		},
		{
			name:        "preinit: can change an existing job",
			phase:       ApplyPythonMutatorPhaseInit,
			updatePath:  dyn.MustPathFromString("resources.jobs.job0.name"),
			deletePath:  dyn.MustPathFromString("resources.jobs.job0.name"),
			insertPath:  dyn.MustPathFromString("resources.jobs.job0.name"),
			deleteError: nil,
			insertError: nil,
			updateError: nil,
		},
		{
			name:        "init: can't delete an existing job",
			phase:       ApplyPythonMutatorPhaseInit,
			deletePath:  dyn.MustPathFromString("resources.jobs.job0"),
			deleteError: fmt.Errorf("unexpected change at \"resources.jobs.job0\" (delete)"),
		},
		{
			name:        "init: can insert a job",
			phase:       ApplyPythonMutatorPhaseInit,
			insertPath:  dyn.MustPathFromString("resources.jobs.job0"),
			insertError: nil,
		},
		{
			name:        "init: can't change include",
			phase:       ApplyPythonMutatorPhaseInit,
			deletePath:  dyn.MustPathFromString("include[0]"),
			insertPath:  dyn.MustPathFromString("include[0]"),
			updatePath:  dyn.MustPathFromString("include[0]"),
			deleteError: fmt.Errorf("unexpected change at \"include[0]\" (delete)"),
			insertError: fmt.Errorf("unexpected change at \"include[0]\" (insert)"),
			updateError: fmt.Errorf("unexpected change at \"include[0]\" (update)"),
		},
	}

	for _, tc := range testCases {
		visitor, err := createOverrideVisitor(context.Background(), tc.phase)

		if err != nil {
			t.Fatalf("create visitor failed: %v", err)
		}

		if tc.updatePath != nil {
			t.Run(tc.name+"-update", func(t *testing.T) {
				out, err := visitor.VisitUpdate(tc.updatePath, left, right)

				if tc.updateError != nil {
					assert.Equal(t, tc.updateError, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, right, out)
				}
			})
		}

		if tc.deletePath != nil {
			t.Run(tc.name+"-delete", func(t *testing.T) {
				err := visitor.VisitDelete(tc.deletePath, left)

				if tc.deleteError != nil {
					assert.Equal(t, tc.deleteError, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}

		if tc.insertPath != nil {
			t.Run(tc.name+"-insert", func(t *testing.T) {
				out, err := visitor.VisitInsert(tc.insertPath, right)

				if tc.insertError != nil {
					assert.Equal(t, tc.insertError, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, right, out)
				}
			})
		}
	}
}

func TestInterpreterPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		assert.Equal(t, "venv\\Scripts\\python3.exe", interpreterPath("venv"))
	} else {
		assert.Equal(t, "venv/bin/python3", interpreterPath("venv"))
	}
}

func withProcessStub(args []string, stdout string) context.Context {
	ctx := context.Background()
	ctx, stub := process.WithStub(ctx)

	stub.WithCallback(func(actual *exec.Cmd) error {
		if reflect.DeepEqual(actual.Args, args) {
			_, err := actual.Stdout.Write([]byte(stdout))

			return err
		} else {
			return fmt.Errorf("unexpected command: %v", actual.Args)
		}
	})

	return ctx
}

func keys[T any](value map[string]T) []string {
	keys := make([]string, 0, len(value))

	for k := range value {
		keys = append(keys, k)
	}

	return keys
}

func loadYaml(name string, content string) *bundle.Bundle {
	v, diag := config.LoadFromBytes(name, []byte(content))

	if diag.Error() != nil {
		panic(diag.Error())
	}

	return &bundle.Bundle{
		Config: *v,
	}
}

func withFakeVEnv(t *testing.T, path string) {
	cwd, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	if err := os.Chdir(t.TempDir()); err != nil {
		panic(err)
	}

	err = os.MkdirAll(filepath.Join(path, "bin"), 0755)

	if err != nil {
		panic(err)
	}

	err = os.WriteFile(filepath.Join(".venv", "bin/python3"), []byte(""), 0755)

	if err != nil {
		panic(err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			panic(err)
		}
	})
}
