package aitools

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// Package-level seams for the interactive pickers of `mcp add` and `mcp remove`. Tests override
// them. The agent picker and the plan confirm are shared with `install` (promptAgentSelection and
// promptProceed in install.go).
var (
	selectProfile          = profile.SelectProfile
	promptServiceSelection = defaultPromptServiceSelection
	promptLocation         = defaultPromptLocation
	promptRemoveSelection  = defaultPromptRemoveSelection
	promptSignInNow        = defaultPromptSignInNow
)

// anotherSchemaValue is the value of the "Another schema..." row. It has no dots, so it cannot be
// a service full name.
const anotherSchemaValue = "(another schema)"

// serviceLister lists the MCP services of a schema.
type serviceLister func(ctx context.Context, location string) ([]mcp.Service, error)

// serviceRow is one row of the service picker: a service of the schema, whether the profile already has it
// installed, and whether it starts checked.
type serviceRow struct {
	service   mcp.Service
	installed bool
	checked   bool
}

// servicePick is what the service picker returns: the rows checked, and whether "Another schema..."
// was checked.
type servicePick struct {
	services []mcp.Service
	another  bool
}

// profileChoice is the cause of the failure that resolveProfile returns when several workspace profiles exist and none
// is the default. It holds those profiles, so that the picker offers the same ones.
type profileChoice struct {
	profiles profile.Profiles
}

func (e *profileChoice) Error() string { return "several workspace profiles" }

// promptProfile asks which of the profiles the servers bind to. Ctrl+C reads as the abort of the other pickers.
func promptProfile(ctx context.Context, profiles profile.Profiles) (string, error) {
	name, err := selectProfile(ctx, profile.SelectConfig{
		Label:    "Which Databricks profile should these MCP servers use?",
		Profiles: profiles,
	})
	if errors.Is(err, cmdio.ErrCtrlC) {
		return "", huh.ErrUserAborted
	}
	return name, err
}

// chooseProfile returns the profile for an add, and whether a picker chose it. A terminal resolves the profile as a
// non-interactive run does: --profile, DATABRICKS_CONFIG_PROFILE, the default profile, then the only workspace profile.
// The picker runs only where that stops with several profiles.
func chooseProfile(ctx context.Context, flagValue string, interactive bool) (string, bool, error) {
	name, err := resolveProfile(ctx, flagValue)
	if err == nil || !interactive {
		return name, false, err
	}
	choice, ok := errors.AsType[*profileChoice](err)
	if !ok {
		return name, false, err
	}
	name, err = promptProfile(ctx, choice.profiles)
	return name, true, err
}

// mcpAgentChoices returns the agent picker rows: the MCP-capable agents installed for MCP, all checked. It also returns
// the MCP-capable agents the picker does not list, each with the reason when its check failed, as a named agent reports it.
func mcpAgentChoices(ctx context.Context) (choices []agentChoice, notFound []string) {
	for _, a := range agents.MCPAgents() {
		installed, err := a.InstalledForMCP(ctx)
		switch {
		case err != nil:
			notFound = append(notFound, fmt.Sprintf("%s (%s)", a.DisplayName, err))
		case !installed:
			notFound = append(notFound, a.DisplayName)
		default:
			choices = append(choices, agentChoice{agent: a, label: a.DisplayName, preselect: true})
		}
	}
	return choices, notFound
}

// pickServicesInteractively lists the services of location and returns the ones the user checks.
// "Another schema..." asks for a catalog and schema and lists that one next; an empty answer stops browsing and keeps
// the picks. A schema is listed once per session, and a schema shown again starts with its checks, which the new page
// replaces. A schema that cannot be listed after the first one is reported and asked for again, with the checks kept.
// Services already installed for the profile are rows too, unchecked and marked installed, so that they can be added to
// another agent; they are also named above the picker.
func pickServicesInteractively(ctx context.Context, b addBinding, location string, state *mcp.State, list serviceLister) ([]mcp.Service, error) {
	// checks holds the services checked on each schema page, listings the services of each schema listed, and schemas
	// the pages in the order shown.
	checks := map[string][]mcp.Service{}
	listings := map[string][]mcp.Service{}
	var schemas []string
	for {
		services, listed := listings[location]
		if !listed {
			var err error
			if services, err = list(ctx, location); err != nil {
				if len(schemas) == 0 {
					return nil, err
				}
				cmdio.LogString(ctx, fmt.Sprintf("Cannot list MCP servers in %s: %s", location, err))
				if location, err = promptLocation(); err != nil {
					return nil, err
				}
				if location == "" {
					break
				}
				continue
			}
			listings[location] = services
			cmdio.LogString(ctx, fmt.Sprintf("Discovering MCP servers in %s on profile %s... %d found.", location, b.profile, len(services)))
		}
		// A page shown first starts with the recommended services that are not installed checked; a page shown
		// again starts with its checks.
		firstVisit := !slices.Contains(schemas, location)
		var installed []string
		rows := make([]serviceRow, 0, len(services))
		for _, s := range services {
			// An entry this profile cannot bind through, as one recorded under another config file, is not shown as
			// installed. The check of the picked services reports it, and one service's entry does not stop the list.
			entry, err := mcp.FindEntry(ctx, state, s.FullName, b.profile, b.configFile)
			isInstalled := err == nil && entry != nil
			if isInstalled {
				installed = append(installed, s.Leaf)
			}
			// A row whose entry cannot be read is refused by the run, so it is not checked.
			checked := err == nil && !isInstalled && s.Known != nil && s.Known.Recommended
			if !firstVisit {
				checked = slices.ContainsFunc(checks[location], func(p mcp.Service) bool { return p.FullName == s.FullName })
			}
			rows = append(rows, serviceRow{service: s, installed: isInstalled, checked: checked})
		}
		if len(installed) > 0 {
			cmdio.LogString(ctx, fmt.Sprintf("Already installed for profile %s: %s", b.profile, strings.Join(installed, ", ")))
		}
		pick, err := promptServiceSelection(location, rows)
		if err != nil {
			return nil, err
		}
		if firstVisit {
			schemas = append(schemas, location)
		}
		checks[location] = pick.services
		if !pick.another {
			break
		}
		if location, err = promptLocation(); err != nil {
			return nil, err
		}
		if location == "" {
			break
		}
	}

	var picked []mcp.Service
	for _, schema := range schemas {
		picked = append(picked, checks[schema]...)
	}
	if len(picked) == 0 {
		return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "select at least one MCP service")
	}
	return picked, nil
}

// typedLocation returns a catalog and schema typed by the user in the form --location takes: trimmed and
// in lowercase.
func typedLocation(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// serviceChoiceLabel is a service picker row: the leaf name, its description and its tags. The
// CLI's own description wins for a known system.ai service; otherwise the API comment is shown.
func serviceChoiceLabel(s mcp.Service) string {
	desc := s.Comment
	var tags []string
	if k := s.Known; k != nil {
		desc = k.Summary
		if k.Recommended {
			tags = append(tags, "recommended")
		}
		if k.Beta {
			tags = append(tags, "beta")
		}
	}
	if s.SignInRequired {
		tags = append(tags, "sign-in required")
	}
	label := s.Leaf
	if desc != "" {
		label += "  " + desc
	}
	if len(tags) > 0 {
		label += "  (" + strings.Join(tags, ", ") + ")"
	}
	return label
}

// serviceRowLabel is the label of a service picker row. A service the profile has installed is marked so.
func serviceRowLabel(r serviceRow) string {
	if r.installed {
		return serviceChoiceLabel(r.service) + "  (installed)"
	}
	return serviceChoiceLabel(r.service)
}

// defaultPromptServiceSelection is the huh multi-select over the rows of a schema.
func defaultPromptServiceSelection(location string, rows []serviceRow) (servicePick, error) {
	options := make([]huh.Option[string], 0, len(rows)+1)
	for _, r := range rows {
		options = append(options, huh.NewOption(serviceRowLabel(r), r.service.FullName).Selected(r.checked))
	}
	options = append(options, huh.NewOption("Another schema...", anotherSchemaValue))

	var selected []string
	err := huh.NewMultiSelect[string]().
		Title("Select MCP servers in " + location).
		Description(multiSelectHelp).
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return servicePick{}, err
	}

	var pick servicePick
	for _, name := range selected {
		if name == anotherSchemaValue {
			pick.another = true
			continue
		}
		i := slices.IndexFunc(rows, func(r serviceRow) bool { return r.service.FullName == name })
		pick.services = append(pick.services, rows[i].service)
	}
	return pick, nil
}

// defaultPromptLocation asks for the catalog and schema to list next.
func defaultPromptLocation() (string, error) {
	var location string
	err := huh.NewInput().
		Title("Catalog and schema to list MCP servers from (leave empty to stop)").
		Value(&location).
		Validate(func(s string) error {
			if typed := typedLocation(s); typed != "" && validateLocation(typed) != nil {
				return errors.New("enter <catalog>.<schema>")
			}
			return nil
		}).
		Run()
	return typedLocation(location), err
}

// chooseRemovals runs the remove picker over the installed servers for targets: only the servers installed for the
// named agents, or every server when targets is nil. Only the agents named with --agents lose a pick, so the picks
// need no agent check. offered is false when no server is offered, and then nothing is asked.
func chooseRemovals(state *mcp.State, targets []*agents.Agent) (picks []mcp.Entry, offered bool, err error) {
	var entries []mcp.Entry
	for _, e := range state.Servers {
		if targets == nil || slices.ContainsFunc(targets, func(a *agents.Agent) bool {
			_, ok := e.Agents[a.Name]
			return ok
		}) {
			entries = append(entries, e)
		}
	}
	if len(entries) == 0 {
		return nil, false, nil
	}
	picks, err = promptRemoveSelection(entries, targets)
	return picks, true, err
}

// removeChoiceLabel is a remove picker row: server name, service, profile and the agents the pick is removed from,
// which are the named agents when --agents is set.
func removeChoiceLabel(e mcp.Entry, targets []*agents.Agent) string {
	var names []string
	for name := range e.Agents {
		if targets == nil || slices.ContainsFunc(targets, func(a *agents.Agent) bool { return a.Name == name }) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return fmt.Sprintf("%s  %s  %s  %s", e.Name, e.Service, e.Profile, mcpAgentList(names))
}

// defaultPromptRemoveSelection is the huh multi-select over the installed servers.
func defaultPromptRemoveSelection(entries []mcp.Entry, targets []*agents.Agent) ([]mcp.Entry, error) {
	options := make([]huh.Option[int], 0, len(entries))
	for i, e := range entries {
		options = append(options, huh.NewOption(removeChoiceLabel(e, targets), i))
	}

	var selected []int
	err := huh.NewMultiSelect[int]().
		Title("Select MCP servers to remove").
		Description(multiSelectHelp).
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return nil, err
	}

	chosen := make([]mcp.Entry, 0, len(selected))
	for _, i := range selected {
		chosen = append(chosen, entries[i])
	}
	return chosen, nil
}

// defaultPromptSignInNow asks whether to start the sign-in of a service now. The default is yes. The sign-in names the
// profile once it starts, so the question does not.
func defaultPromptSignInNow(service string) (bool, error) {
	yes := true
	err := huh.NewConfirm().
		Title(service + " needs a one-time sign-in. Sign in now?").
		Value(&yes).
		Run()
	if err != nil {
		return false, err
	}
	return yes, nil
}
