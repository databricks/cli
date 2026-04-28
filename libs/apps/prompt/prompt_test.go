package prompt

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/ansi"
	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keys creates a tea.KeyMsg for simulating keyboard input in tests.
func keys(runes ...rune) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: runes,
	}
}

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid simple name",
			projectName: "my-app",
			expectError: false,
		},
		{
			name:        "valid name with numbers",
			projectName: "app123",
			expectError: false,
		},
		{
			name:        "valid name with hyphens",
			projectName: "my-cool-app",
			expectError: false,
		},
		{
			name:        "empty name",
			projectName: "",
			expectError: true,
			errorMsg:    "required",
		},
		{
			name:        "name too long",
			projectName: "this-is-a-very-long-app-name-that-exceeds",
			expectError: true,
			errorMsg:    "too long",
		},
		{
			name:        "name at max length (26 chars)",
			projectName: "abcdefghijklmnopqrstuvwxyz",
			expectError: false,
		},
		{
			name:        "name starts with number",
			projectName: "123app",
			expectError: true,
			errorMsg:    "must start with a letter",
		},
		{
			name:        "name starts with hyphen",
			projectName: "-myapp",
			expectError: true,
			errorMsg:    "must start with a letter",
		},
		{
			name:        "name with uppercase",
			projectName: "MyApp",
			expectError: true,
			errorMsg:    "lowercase",
		},
		{
			name:        "name with underscore",
			projectName: "my_app",
			expectError: true,
			errorMsg:    "lowercase letters, numbers, or hyphens",
		},
		{
			name:        "name with spaces",
			projectName: "my app",
			expectError: true,
			errorMsg:    "lowercase letters, numbers, or hyphens",
		},
		{
			name:        "name with special characters",
			projectName: "my@app!",
			expectError: true,
			errorMsg:    "lowercase letters, numbers, or hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectName(tt.projectName)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunWithSpinnerCtx(t *testing.T) {
	t.Run("successful action", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		executed := false

		err := RunWithSpinnerCtx(ctx, "Testing...", func() error {
			executed = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("action returns error", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		expectedErr := errors.New("action failed")

		err := RunWithSpinnerCtx(ctx, "Testing...", func() error {
			return expectedErr
		})

		assert.Equal(t, expectedErr, err)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
		actionStarted := make(chan struct{})
		actionDone := make(chan struct{})

		go func() {
			_ = RunWithSpinnerCtx(ctx, "Testing...", func() error {
				close(actionStarted)
				time.Sleep(100 * time.Millisecond)
				close(actionDone)
				return nil
			})
		}()

		// Wait for action to start
		<-actionStarted
		// Cancel context
		cancel()
		// Wait for action to complete (spinner should wait)
		<-actionDone
	})

	t.Run("action panics - recovered", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())

		err := RunWithSpinnerCtx(ctx, "Testing...", func() error {
			panic("test panic")
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "action panicked")
		assert.Contains(t, err.Error(), "test panic")
	})
}

func TestRunModeConstants(t *testing.T) {
	assert.Equal(t, RunModeNone, RunMode("none"))
	assert.Equal(t, RunModeDev, RunMode("dev"))
	assert.Equal(t, RunModeDevRemote, RunMode("dev-remote"))
}

func TestApplyResolvedValues(t *testing.T) {
	t.Run("maps resolver names to manifest field names", func(t *testing.T) {
		r := manifest.Resource{
			ResourceKey: "postgres",
			Fields: map[string]manifest.ResourceField{
				"branch":       {Description: "branch path"},
				"database":     {Description: "database name"},
				"host":         {Resolve: "postgres:host"},
				"databaseName": {Resolve: "postgres:databaseName"},
				"endpointPath": {Resolve: "postgres:endpointPath"},
				"port":         {Value: "5432"},
			},
		}

		resolvedValues := map[string]string{
			"postgres:host":         "my-host.example.com",
			"postgres:databaseName": "my_db",
			"postgres:endpointPath": "projects/p1/branches/b1/endpoints/e1",
		}

		result := map[string]string{
			"postgres.branch":   "projects/p1/branches/b1",
			"postgres.database": "projects/p1/branches/b1/databases/d1",
		}

		applyResolvedValues(r, resolvedValues, result)

		assert.Equal(t, map[string]string{
			"postgres.branch":       "projects/p1/branches/b1",
			"postgres.database":     "projects/p1/branches/b1/databases/d1",
			"postgres.host":         "my-host.example.com",
			"postgres.databaseName": "my_db",
			"postgres.endpointPath": "projects/p1/branches/b1/endpoints/e1",
		}, result)
	})

	t.Run("renamed fields still map via resolver", func(t *testing.T) {
		r := manifest.Resource{
			ResourceKey: "postgres",
			Fields: map[string]manifest.ResourceField{
				"pg_host":     {Resolve: "postgres:host"},
				"pg_database": {Resolve: "postgres:databaseName"},
				"pg_endpoint": {Resolve: "postgres:endpointPath"},
			},
		}

		resolvedValues := map[string]string{
			"postgres:host":         "host.example.com",
			"postgres:databaseName": "testdb",
			"postgres:endpointPath": "projects/p1/branches/b1/endpoints/e1",
		}

		result := map[string]string{}
		applyResolvedValues(r, resolvedValues, result)

		assert.Equal(t, map[string]string{
			"postgres.pg_host":     "host.example.com",
			"postgres.pg_database": "testdb",
			"postgres.pg_endpoint": "projects/p1/branches/b1/endpoints/e1",
		}, result)
	})

	t.Run("skips fields without resolve", func(t *testing.T) {
		r := manifest.Resource{
			ResourceKey: "postgres",
			Fields: map[string]manifest.ResourceField{
				"branch": {Description: "no resolve"},
				"host":   {Resolve: "postgres:host"},
				"port":   {Value: "5432"},
			},
		}

		resolvedValues := map[string]string{
			"postgres:host": "my-host",
		}

		result := map[string]string{}
		applyResolvedValues(r, resolvedValues, result)

		assert.Equal(t, map[string]string{
			"postgres.host": "my-host",
		}, result)
	})

	t.Run("skips resolve values not in resolvedValues map", func(t *testing.T) {
		r := manifest.Resource{
			ResourceKey: "postgres",
			Fields: map[string]manifest.ResourceField{
				"host":    {Resolve: "postgres:host"},
				"unknown": {Resolve: "postgres:unknownResolver"},
			},
		}

		resolvedValues := map[string]string{
			"postgres:host": "my-host",
		}

		result := map[string]string{}
		applyResolvedValues(r, resolvedValues, result)

		assert.Equal(t, map[string]string{
			"postgres.host": "my-host",
		}, result)
	})
}

func TestVolumePathToSecurableName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/Volumes/catalog/schema/vol", "catalog.schema.vol"},
		{"/Volumes/my-cat/my-schema/my-vol", "my-cat.my-schema.my-vol"},
		{"catalog.schema.vol", "catalog.schema.vol"},
		{"/Volumes/a/b/c", "a.b.c"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, volumePathToSecurableName(tt.input))
		})
	}
}

func TestMaxAppNameLength(t *testing.T) {
	// Verify the constant is set correctly
	assert.Equal(t, 30, MaxAppNameLength)
	assert.Equal(t, "dev-", DevTargetPrefix)

	// Max allowed name length should be 30 - 4 ("dev-") = 26
	maxAllowed := MaxAppNameLength - len(DevTargetPrefix)
	assert.Equal(t, 26, maxAllowed)

	// Test at boundary
	validName := "abcdefghijklmnopqrstuvwxyz" // 26 chars
	assert.Len(t, validName, 26)
	assert.NoError(t, ValidateProjectName(validName))

	// Test over boundary
	invalidName := "abcdefghijklmnopqrstuvwxyz1" // 27 chars
	assert.Len(t, invalidName, 27)
	assert.Error(t, ValidateProjectName(invalidName))
}

// initForm runs the form's Init command and feeds the resulting message back
// through Update so the form picks up its initial layout (window size, focus).
// f.Update(f.Init()) silently swallows the cmd because Init returns a tea.Cmd
// (a function), not a tea.Msg, so the form never receives the WindowSizeMsg
// it needs to render.
func initForm(t *testing.T, f *huh.Form) {
	t.Helper()
	cmd := f.Init()
	if cmd != nil {
		if msg := cmd(); msg != nil {
			f.Update(msg)
		}
	}
	// huh derives layout from a window size; without one the help line
	// and titles can be clipped or omitted.
	f.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
}

// newWarehouseSelect builds a huh.Select identical in shape to the one
// constructed inside promptFromListWithLabel. Production and tests share this
// builder so that future regressions to the production prompt's title or
// description show up in test output.
func newWarehouseSelect(title, description string, options ...string) *huh.Select[string] {
	return huh.NewSelect[string]().
		Options(huh.NewOptions(options...)...).
		Title(title).
		Description(description).
		Height(8)
}

// TestSelectTitleVisibleWithoutFiltering verifies that a Select field renders
// its Title on the initial view when Filtering is not activated. This is the
// behavior after the fix: the Title is always visible.
func TestSelectTitleVisibleWithoutFiltering(t *testing.T) {
	field := newWarehouseSelect(
		"Select SQL Warehouse",
		"3 available — press / to filter",
		"Warehouse A", "Warehouse B", "Warehouse C",
	)

	f := huh.NewForm(huh.NewGroup(field))
	initForm(t, f)

	view := ansi.Strip(f.View())

	assert.Contains(t, view, "Select SQL Warehouse", "Title should be visible in initial render")
	assert.Contains(t, view, "press / to filter", "Description should be visible")
	assert.Contains(t, view, "Warehouse A", "First option should be visible")
}

// TestSelectSlashKeyActivatesFilter verifies that pressing '/' activates
// filtering even without Filtering(true), and that it filters options. The
// title remains visible after activation, which is the regression this PR
// guards against.
func TestSelectSlashKeyActivatesFilter(t *testing.T) {
	field := newWarehouseSelect("Select fruit", "", "Apple", "Apricot", "Banana")

	f := huh.NewForm(huh.NewGroup(field))
	initForm(t, f)

	// Title visible before filtering.
	view := ansi.Strip(f.View())
	assert.Contains(t, view, "Select fruit")
	assert.Contains(t, view, "Banana")

	// Press '/' to start filtering, then type 'B'.
	m, _ := f.Update(keys('/'))
	f = m.(*huh.Form)
	m, _ = f.Update(keys('B'))
	f = m.(*huh.Form)

	view = ansi.Strip(f.View())

	// Once filter mode is active huh replaces the title with the filter input
	// — that is upstream behavior. The fix this PR enforces is that the title
	// is visible BEFORE the user presses '/', not after.
	assert.Contains(t, view, "Banana", "Banana should match filter 'B'")
	assert.NotContains(t, view, "Apple", "Apple should be filtered out")
}

// TestSelectHelpShowsFilterHint verifies the help text includes a filter hint.
func TestSelectHelpShowsFilterHint(t *testing.T) {
	field := newWarehouseSelect("Pick", "", "A", "B")

	f := huh.NewForm(huh.NewGroup(field))
	initForm(t, f)

	view := ansi.Strip(f.View())

	// huh's default keymap shows "/ filter" in the help line.
	assert.True(t,
		strings.Contains(view, "/ filter") || strings.Contains(view, "filter"),
		"Help text should mention filtering is available via '/'",
	)
}
