package aitools

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPickServicesInteractivelyKeepsChecksAcrossSchemas(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	state := &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood"}}}
	schemas := map[string][]mcp.Service{
		"system.ai":       {{FullName: "system.ai.github", Leaf: "github"}, {FullName: "system.ai.dbsql", Leaf: "dbsql"}},
		"analytics.tools": {{FullName: "analytics.tools.search", Leaf: "search"}},
	}
	var listed []string
	list := func(_ context.Context, location string) ([]mcp.Service, error) {
		listed = append(listed, location)
		return schemas[location], nil
	}

	var rowsPerPage [][]string
	origPick := promptServiceSelection
	t.Cleanup(func() { promptServiceSelection = origPick })
	promptServiceSelection = func(location string, rows []serviceRow) (servicePick, error) {
		var names []string
		for _, r := range rows {
			names = append(names, r.service.FullName)
			// The installed github row is shown, marked installed and unchecked.
			if r.service.FullName == "system.ai.github" {
				assert.True(t, r.installed)
				assert.False(t, r.checked)
			}
		}
		rowsPerPage = append(rowsPerPage, names)
		if location == "system.ai" {
			// Checks dbsql, then asks for another schema.
			return servicePick{services: []mcp.Service{rows[1].service}, another: true}, nil
		}
		return servicePick{services: []mcp.Service{rows[0].service}}, nil
	}
	origLocation := promptLocation
	t.Cleanup(func() { promptLocation = origLocation })
	promptLocation = func() (string, error) { return "analytics.tools", nil }

	picked, err := pickServicesInteractively(ctx, defaultBinding(t, "logfood"), "system.ai", state, list)
	require.NoError(t, err)
	assert.Equal(t, []string{"system.ai", "analytics.tools"}, listed)
	assert.Equal(t, [][]string{{"system.ai.github", "system.ai.dbsql"}, {"analytics.tools.search"}}, rowsPerPage)
	var names []string
	for _, s := range picked {
		names = append(names, s.FullName)
	}
	assert.Equal(t, []string{"system.ai.dbsql", "analytics.tools.search"}, names)
}

// TestPickServicesInteractivelyRevisitShowsAndReplacesChecks checks that a schema shown again starts with the
// services checked on it before, and that its new page replaces them, so an unchecked service is dropped.
func TestPickServicesInteractivelyRevisitShowsAndReplacesChecks(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	schemas := map[string][]mcp.Service{
		"system.ai":       {{FullName: "system.ai.github", Leaf: "github"}, {FullName: "system.ai.dbsql", Leaf: "dbsql"}},
		"analytics.tools": {{FullName: "analytics.tools.search", Leaf: "search"}},
	}
	listed := map[string]int{}
	list := func(_ context.Context, location string) ([]mcp.Service, error) {
		listed[location]++
		return schemas[location], nil
	}

	// Each page answers with the rows it is given, checked or not, and the schema sequence the user typed.
	type page struct {
		location string
		checked  map[string]bool
		pick     []string
		another  bool
	}
	pages := []page{
		{location: "system.ai", checked: map[string]bool{"system.ai.github": false, "system.ai.dbsql": false}, pick: []string{"system.ai.dbsql"}, another: true},
		{location: "analytics.tools", checked: map[string]bool{"analytics.tools.search": false}, pick: []string{"analytics.tools.search"}, another: true},
		// Back on system.ai, dbsql is shown checked; the user unchecks it and checks github.
		{location: "system.ai", checked: map[string]bool{"system.ai.github": false, "system.ai.dbsql": true}, pick: []string{"system.ai.github"}, another: true},
		// Back on analytics.tools, search is shown checked and the user finishes there.
		{location: "analytics.tools", checked: map[string]bool{"analytics.tools.search": true}, pick: []string{"analytics.tools.search"}},
	}
	origPick := promptServiceSelection
	t.Cleanup(func() { promptServiceSelection = origPick })
	promptServiceSelection = func(location string, rows []serviceRow) (servicePick, error) {
		require.NotEmpty(t, pages, "more pages than expected")
		want := pages[0]
		pages = pages[1:]
		require.Equal(t, want.location, location)
		for _, r := range rows {
			assert.Equal(t, want.checked[r.service.FullName], r.checked, r.service.FullName)
		}
		var picked []mcp.Service
		for _, name := range want.pick {
			for _, s := range schemas[location] {
				if s.FullName == name {
					picked = append(picked, s)
				}
			}
		}
		return servicePick{services: picked, another: want.another}, nil
	}
	locations := []string{"analytics.tools", "system.ai", "analytics.tools"}
	origLocation := promptLocation
	t.Cleanup(func() { promptLocation = origLocation })
	promptLocation = func() (string, error) {
		next := locations[0]
		locations = locations[1:]
		return next, nil
	}

	picked, err := pickServicesInteractively(ctx, defaultBinding(t, "logfood"), "system.ai", &mcp.State{}, list)
	require.NoError(t, err)
	require.Empty(t, pages, "every page was shown")
	var names []string
	for _, s := range picked {
		names = append(names, s.FullName)
	}
	assert.Equal(t, []string{"system.ai.github", "analytics.tools.search"}, names)
	// A schema shown again is not listed again.
	assert.Equal(t, map[string]int{"system.ai": 1, "analytics.tools": 1}, listed)
}

// TestPickServicesInteractivelyEmptyLocationStopsAndKeepsPicks checks that an empty answer to "Another schema" stops
// browsing: the picks of every page so far are kept, and no other schema is listed.
func TestPickServicesInteractivelyEmptyLocationStopsAndKeepsPicks(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	list := func(_ context.Context, location string) ([]mcp.Service, error) {
		require.Equal(t, "system.ai", location, "no schema is listed after the empty answer")
		return []mcp.Service{{FullName: "system.ai.github", Leaf: "github"}, {FullName: "system.ai.dbsql", Leaf: "dbsql"}}, nil
	}
	origPick := promptServiceSelection
	t.Cleanup(func() { promptServiceSelection = origPick })
	promptServiceSelection = func(_ string, rows []serviceRow) (servicePick, error) {
		return servicePick{services: []mcp.Service{rows[0].service}, another: true}, nil
	}
	origLocation := promptLocation
	t.Cleanup(func() { promptLocation = origLocation })
	asked := 0
	promptLocation = func() (string, error) {
		asked++
		return "", nil
	}

	picked, err := pickServicesInteractively(ctx, defaultBinding(t, "logfood"), "system.ai", &mcp.State{}, list)
	require.NoError(t, err)
	assert.Equal(t, 1, asked)
	require.Len(t, picked, 1)
	assert.Equal(t, "system.ai.github", picked[0].FullName)
}

// TestPickServicesInteractivelyKeepsPicksWhenAnotherSchemaFails checks that a schema that cannot be listed is
// reported and asked for again, and that the services picked before are kept. A failure on the first schema
// ends the run, since nothing is picked yet.
func TestPickServicesInteractivelyKeepsPicksWhenAnotherSchemaFails(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	list := func(_ context.Context, location string) ([]mcp.Service, error) {
		switch location {
		case "system.ai":
			return []mcp.Service{{FullName: "system.ai.github", Leaf: "github"}}, nil
		case "analytics.tools":
			return []mcp.Service{{FullName: "analytics.tools.search", Leaf: "search"}}, nil
		}
		return nil, errors.New("schema not found")
	}
	origPick := promptServiceSelection
	t.Cleanup(func() { promptServiceSelection = origPick })
	promptServiceSelection = func(_ string, rows []serviceRow) (servicePick, error) {
		return servicePick{services: []mcp.Service{rows[0].service}, another: rows[0].service.FullName == "system.ai.github"}, nil
	}
	// The user types a typo first, then the schema that exists. The prompt returns the location trimmed and in lowercase.
	typed := []string{"analytcs.tools", "analytics.tools"}
	origLocation := promptLocation
	t.Cleanup(func() { promptLocation = origLocation })
	promptLocation = func() (string, error) {
		next := typed[0]
		typed = typed[1:]
		return next, nil
	}

	picked, err := pickServicesInteractively(ctx, defaultBinding(t, "logfood"), "system.ai", &mcp.State{}, list)
	require.NoError(t, err)
	var names []string
	for _, s := range picked {
		names = append(names, s.FullName)
	}
	assert.Equal(t, []string{"system.ai.github", "analytics.tools.search"}, names)

	// A failure on the first schema is returned, since nothing is picked yet.
	_, err = pickServicesInteractively(ctx, defaultBinding(t, "logfood"), "missing.schema", &mcp.State{}, list)
	assert.EqualError(t, err, "schema not found")
}

// TestPickServicesInteractivelyEntryUnderOtherConfigFile checks that a service recorded under another config file does
// not stop the picker for the other services: its row is not marked installed, and the check of the picked services
// reports the binding.
func TestPickServicesInteractivelyEntryUnderOtherConfigFile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	state := &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", ConfigFile: filepath.Join(t.TempDir(), "other.cfg"),
	}}}
	list := func(context.Context, string) ([]mcp.Service, error) {
		return []mcp.Service{{FullName: "system.ai.github", Leaf: "github"}, {FullName: "system.ai.dbsql", Leaf: "dbsql"}}, nil
	}
	origPick := promptServiceSelection
	t.Cleanup(func() { promptServiceSelection = origPick })
	promptServiceSelection = func(_ string, rows []serviceRow) (servicePick, error) {
		require.Len(t, rows, 2)
		assert.False(t, rows[0].installed)
		assert.False(t, rows[1].installed)
		return servicePick{services: []mcp.Service{rows[1].service}}, nil
	}

	picked, err := pickServicesInteractively(ctx, defaultBinding(t, "logfood"), "system.ai", state, list)
	require.NoError(t, err)
	require.Len(t, picked, 1)
	assert.Equal(t, "system.ai.dbsql", picked[0].FullName)
}

// TestPickServicesInteractivelyUncheckableEntryIsNotChecked checks that a recommended service whose entry cannot be read
// is not checked: the run refuses it, so the picker must not offer it as a pick.
func TestPickServicesInteractivelyUncheckableEntryIsNotChecked(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	state := &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", ConfigFile: filepath.Join(t.TempDir(), "other.cfg"),
	}}}
	list := func(context.Context, string) ([]mcp.Service, error) {
		return []mcp.Service{
			{FullName: "system.ai.github", Leaf: "github", Known: &mcp.KnownService{Recommended: true}},
			{FullName: "system.ai.dbsql", Leaf: "dbsql", Known: &mcp.KnownService{Recommended: true}},
		}, nil
	}
	origPick := promptServiceSelection
	t.Cleanup(func() { promptServiceSelection = origPick })
	promptServiceSelection = func(_ string, rows []serviceRow) (servicePick, error) {
		require.Len(t, rows, 2)
		assert.False(t, rows[0].checked, "the entry under another config file must not start checked")
		assert.True(t, rows[1].checked)
		return servicePick{services: []mcp.Service{rows[1].service}}, nil
	}

	_, err := pickServicesInteractively(ctx, defaultBinding(t, "logfood"), "system.ai", state, list)
	require.NoError(t, err)
}

// TestPickServicesInteractivelyAnnouncesEachListingOnce checks that the discovery line is printed for each listing that
// runs, and not again when a schema is shown again.
func TestPickServicesInteractivelyAnnouncesEachListingOnce(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	schemas := map[string][]mcp.Service{
		"system.ai":       {{FullName: "system.ai.github", Leaf: "github"}},
		"analytics.tools": {{FullName: "analytics.tools.search", Leaf: "search"}},
	}
	list := func(_ context.Context, location string) ([]mcp.Service, error) {
		return schemas[location], nil
	}
	origPick := promptServiceSelection
	t.Cleanup(func() { promptServiceSelection = origPick })
	pages := []struct {
		another bool
	}{{another: true}, {another: true}, {}}
	promptServiceSelection = func(_ string, rows []serviceRow) (servicePick, error) {
		page := pages[0]
		pages = pages[1:]
		return servicePick{services: []mcp.Service{rows[0].service}, another: page.another}, nil
	}
	locations := []string{"analytics.tools", "system.ai"}
	origLocation := promptLocation
	t.Cleanup(func() { promptLocation = origLocation })
	promptLocation = func() (string, error) {
		next := locations[0]
		locations = locations[1:]
		return next, nil
	}

	_, err := pickServicesInteractively(ctx, defaultBinding(t, "logfood"), "system.ai", &mcp.State{}, list)
	require.NoError(t, err)
	assert.Empty(t, pages)
	assert.Equal(t, 2, strings.Count(stderr.String(), "Discovering MCP servers in "), stderr.String())
}

func TestPickServicesInteractivelyNeedsOneService(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	list := func(context.Context, string) ([]mcp.Service, error) {
		return []mcp.Service{{FullName: "system.ai.dbsql", Leaf: "dbsql"}}, nil
	}
	origPick := promptServiceSelection
	t.Cleanup(func() { promptServiceSelection = origPick })
	promptServiceSelection = func(string, []serviceRow) (servicePick, error) {
		return servicePick{}, nil
	}

	_, err := pickServicesInteractively(ctx, defaultBinding(t, "logfood"), "system.ai", &mcp.State{}, list)
	require.Error(t, err)
	assert.Equal(t, "select at least one MCP service", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
}

func TestServiceChoiceLabel(t *testing.T) {
	cases := []struct {
		name    string
		service mcp.Service
		want    string
	}{
		{
			name: "known and recommended",
			service: mcp.Service{Leaf: "genie_one_mcp", Comment: "api text", Known: &mcp.KnownService{
				Summary: "Ask questions about your data", Recommended: true,
			}},
			want: "genie_one_mcp  Ask questions about your data  (recommended)",
		},
		{
			name:    "sign-in required uses the API comment",
			service: mcp.Service{Leaf: "search", Comment: "Search docs", SignInRequired: true},
			want:    "search  Search docs  (sign-in required)",
		},
		{
			name:    "no description",
			service: mcp.Service{Leaf: "bare"},
			want:    "bare",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, serviceChoiceLabel(tc.service))
		})
	}
}

func TestChooseProfileDefaultIsNotAsked(t *testing.T) {
	writeDatabricksCfg(t, "[__settings__]\ndefault_profile = dogfood\n\n[logfood]\nhost = https://logfood.test\n\n[dogfood]\nhost = https://dogfood.test\n")

	origSelect := selectProfile
	t.Cleanup(func() { selectProfile = origSelect })
	selectProfile = func(context.Context, profile.SelectConfig) (string, error) {
		t.Fatal("the picker must not run when the default profile is configured")
		return "", nil
	}

	name, picked, err := chooseProfile(t.Context(), "", true)
	require.NoError(t, err)
	assert.Equal(t, "dogfood", name)
	assert.False(t, picked)
}

func TestChooseProfilePickerListsEveryProfile(t *testing.T) {
	writeDatabricksCfg(t, "[logfood]\nhost = https://logfood.test\n\n[dogfood]\nhost = https://dogfood.test\n")

	origSelect := selectProfile
	t.Cleanup(func() { selectProfile = origSelect })
	var got profile.SelectConfig
	selectProfile = func(_ context.Context, cfg profile.SelectConfig) (string, error) {
		got = cfg
		return "logfood", nil
	}

	name, picked, err := chooseProfile(t.Context(), "", true)
	require.NoError(t, err)
	assert.Equal(t, "logfood", name)
	assert.True(t, picked)
	assert.Len(t, got.Profiles, 2)
}

// TestChooseProfileCtrlCIsTheOtherPickersAbort checks that Ctrl+C at the profile picker aborts as the other pickers do:
// the same error, categorized as USER_ABORTED.
func TestChooseProfileCtrlCIsTheOtherPickersAbort(t *testing.T) {
	writeDatabricksCfg(t, "[logfood]\nhost = https://logfood.test\n\n[dogfood]\nhost = https://dogfood.test\n")

	origSelect := selectProfile
	t.Cleanup(func() { selectProfile = origSelect })
	selectProfile = func(context.Context, profile.SelectConfig) (string, error) {
		return "", cmdio.ErrCtrlC
	}

	_, picked, err := chooseProfile(t.Context(), "", true)
	require.ErrorIs(t, err, huh.ErrUserAborted)
	assert.True(t, picked)
	assert.Equal(t, "user aborted", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryUserAborted, errorCategory(err))
}

func TestChooseProfileSingleProfileIsNotAsked(t *testing.T) {
	writeDatabricksCfg(t, "[logfood]\nhost = https://logfood.test\n")

	origSelect := selectProfile
	t.Cleanup(func() { selectProfile = origSelect })
	selectProfile = func(context.Context, profile.SelectConfig) (string, error) {
		t.Fatal("the picker must not run with one workspace profile")
		return "", nil
	}

	name, picked, err := chooseProfile(t.Context(), "", true)
	require.NoError(t, err)
	assert.Equal(t, "logfood", name)
	assert.False(t, picked)
}

func TestChooseProfileNoProfiles(t *testing.T) {
	writeDatabricksCfg(t, "")
	_, _, err := chooseProfile(t.Context(), "", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no workspace profiles in")
}

func TestChooseProfileWithoutTerminalDoesNotPrompt(t *testing.T) {
	writeDatabricksCfg(t, "[logfood]\nhost = https://logfood.test\n\n[dogfood]\nhost = https://dogfood.test\n")

	origSelect := selectProfile
	t.Cleanup(func() { selectProfile = origSelect })
	selectProfile = func(context.Context, profile.SelectConfig) (string, error) {
		t.Fatal("the picker must not run without a terminal")
		return "", nil
	}

	_, _, err := chooseProfile(t.Context(), "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "several workspace profiles exist; pass --profile")

	name, picked, err := chooseProfile(t.Context(), "dogfood", true)
	require.NoError(t, err)
	assert.Equal(t, "dogfood", name)
	assert.False(t, picked)
}

// TestChooseProfileEnvironmentIsAnExplicitChoice checks that DATABRICKS_CONFIG_PROFILE chooses the profile in a
// terminal too, as --profile does: the picker does not run, and the profile resolves as it does without a terminal.
func TestChooseProfileEnvironmentIsAnExplicitChoice(t *testing.T) {
	writeDatabricksCfg(t, "[logfood]\nhost = https://logfood.test\n\n[dogfood]\nhost = https://dogfood.test\n")
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "dogfood")

	origSelect := selectProfile
	t.Cleanup(func() { selectProfile = origSelect })
	selectProfile = func(context.Context, profile.SelectConfig) (string, error) {
		t.Fatal("the picker must not run when DATABRICKS_CONFIG_PROFILE is set")
		return "", nil
	}

	name, picked, err := chooseProfile(t.Context(), "", true)
	require.NoError(t, err)
	assert.Equal(t, "dogfood", name)
	assert.False(t, picked)

	// A variable that names a profile missing from the file is what a non-interactive run gets too.
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "missing")
	interactive, _, err := chooseProfile(t.Context(), "", true)
	require.NoError(t, err)
	plain, _, err := chooseProfile(t.Context(), "", false)
	require.NoError(t, err)
	assert.Equal(t, "missing", interactive)
	assert.Equal(t, plain, interactive)
}

func TestMcpAgentChoicesListInstalledAgents(t *testing.T) {
	withHome(t)
	writeFakeCLI(t, agents.ByName(agents.NameCodex).Binary)
	choices, notFound := mcpAgentChoices(t.Context())

	var listed []string
	for _, c := range choices {
		assert.True(t, c.agent.SupportsMCP(), c.agent.Name)
		assert.True(t, c.preselect, c.agent.Name)
		listed = append(listed, c.agent.Name)
	}
	assert.Contains(t, listed, agents.NameCodex)
	assert.NotContains(t, listed, agents.NameCursor)
	assert.Contains(t, notFound, agents.ByName(agents.NameCursor).DisplayName)
	assert.NotContains(t, notFound, agents.ByName(agents.NameCodex).DisplayName)
}

// TestMcpAgentChoicesFollowInstalledForMCP checks that an agent is listed when InstalledForMCP says so: Codex on PATH
// counts as installed for MCP, even without a ~/.codex directory, and with CODEX_HOME set only that directory counts.
// TestMcpAgentChoicesNamesAgentWhoseCheckFailed checks that an agent whose installed check fails is not offered, and is
// named as not found with that error, as a named agent reports it.
func TestMcpAgentChoicesNamesAgentWhoseCheckFailed(t *testing.T) {
	t.Setenv(env.HomeEnvVar(), "")
	t.Setenv("PATH", t.TempDir())
	for _, name := range agentHomeEnv {
		t.Setenv(name, "")
	}
	ctx := cmdio.MockDiscard(t.Context())
	claude := agents.ByName(agents.NameClaudeCode)
	_, checkErr := claude.InstalledForMCP(ctx)
	require.Error(t, checkErr, "the check needs a home directory")

	choices, notFound := mcpAgentChoices(ctx)
	for _, c := range choices {
		assert.NotEqual(t, claude, c.agent)
	}
	assert.Contains(t, notFound, fmt.Sprintf("%s (%s)", claude.DisplayName, checkErr))
}

func TestMcpAgentChoicesFollowInstalledForMCP(t *testing.T) {
	home := withHome(t)
	writeFakeCLI(t, agents.ByName(agents.NameCodex).Binary)
	assert.True(t, listsAgent(t, agents.NameCodex))

	// The binary on PATH does not count when CODEX_HOME names a directory that is not there, and nothing is created.
	missing := filepath.Join(home, "missing-codex-home")
	t.Setenv(codexHomeEnv, missing)
	assert.False(t, listsAgent(t, agents.NameCodex))
	assert.NoDirExists(t, missing)
}

// listsAgent reports whether the agent picker lists the agent named name.
func listsAgent(t *testing.T, name string) bool {
	t.Helper()
	choices, _ := mcpAgentChoices(t.Context())
	return slices.ContainsFunc(choices, func(c agentChoice) bool { return c.agent.Name == name })
}

func TestChooseRemovals(t *testing.T) {
	origRemove := promptRemoveSelection
	t.Cleanup(func() { promptRemoveSelection = origRemove })
	promptRemoveSelection = func([]mcp.Entry, []*agents.Agent) ([]mcp.Entry, error) {
		t.Fatal("the picker must not run with nothing installed")
		return nil, nil
	}
	got, offered, err := chooseRemovals(&mcp.State{}, nil)
	require.NoError(t, err)
	assert.False(t, offered)
	assert.Empty(t, got)

	state := &mcp.State{Servers: []mcp.Entry{
		{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood"},
		{Name: "github-dogfood", Service: "system.ai.github", Profile: "dogfood"},
	}}
	promptRemoveSelection = func(entries []mcp.Entry, _ []*agents.Agent) ([]mcp.Entry, error) {
		assert.Len(t, entries, 2)
		return entries[1:], nil
	}
	got, offered, err = chooseRemovals(state, nil)
	require.NoError(t, err)
	assert.True(t, offered)
	require.Len(t, got, 1)
	assert.Equal(t, "github-dogfood", got[0].Name)

	// --agents narrows the picker to the servers installed for those agents.
	state.Servers[0].Agents = map[string]mcp.AgentRecord{agents.NameCodex: {}}
	state.Servers[1].Agents = map[string]mcp.AgentRecord{agents.NameClaudeCode: {}}
	promptRemoveSelection = func(entries []mcp.Entry, _ []*agents.Agent) ([]mcp.Entry, error) {
		require.Len(t, entries, 1)
		assert.Equal(t, "github-logfood", entries[0].Name)
		return entries, nil
	}
	got, offered, err = chooseRemovals(state, []*agents.Agent{agents.ByName(agents.NameCodex)})
	require.NoError(t, err)
	assert.True(t, offered)
	require.Len(t, got, 1)
	assert.Equal(t, "github-logfood", got[0].Name)

	// No server for those agents: nothing is offered.
	_, offered, err = chooseRemovals(state, []*agents.Agent{agents.ByName(agents.NameCursor)})
	require.NoError(t, err)
	assert.False(t, offered)
}

func TestRemoveChoiceLabel(t *testing.T) {
	entry := mcp.Entry{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood",
		Agents: map[string]mcp.AgentRecord{agents.NameCodex: {}, agents.NameClaudeCode: {}},
	}
	assert.Equal(t, "github-logfood  system.ai.github  logfood  Claude Code, Codex CLI", removeChoiceLabel(entry, nil))

	// With --agents, the label names only the agents the pick is removed from.
	codex := []*agents.Agent{agents.ByName(agents.NameCodex)}
	assert.Equal(t, "github-logfood  system.ai.github  logfood  Codex CLI", removeChoiceLabel(entry, codex))
}
