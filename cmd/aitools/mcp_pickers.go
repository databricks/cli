package aitools

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"golang.org/x/term"
)

// Package-level seams for the interactive pickers of `mcp add` and `mcp remove`. Tests override
// them.
var (
	selectProfile          = profile.SelectProfile
	promptMCPAgents        = defaultPromptMCPAgents
	promptServiceSelection = defaultPromptServiceSelection
	promptLocation         = defaultPromptLocation
	promptRemoveSelection  = defaultPromptRemoveSelection
	promptSignInNow        = defaultPromptSignInNow
	promptMCPProceed       = defaultPromptMCPProceed
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

// profilePickerActive and profilePickerInactive are the rows of the profile picker. Both show the host, so that every
// profile reads alike, and the active row has the ▸ marker and underline of the cmdio select. No row is marked as the
// default, since the picker runs only when there is no default profile.
const (
	profilePickerActive   = `{{ "▸" | bold }} {{ .Name | underline }}{{ if .Host }} ({{ .Host | faint }}){{ end }}`
	profilePickerInactive = `  {{ .Name }}{{ if .Host }} ({{ .Host | faint }}){{ end }}`
)

// promptProfile asks which of the profiles the servers bind to. Ctrl+C reads as the abort of the other pickers.
func promptProfile(ctx context.Context, profiles profile.Profiles) (string, error) {
	name, err := selectProfile(ctx, profile.SelectConfig{
		Label:            "Which Databricks profile should these MCP servers use?",
		Profiles:         profiles,
		ActiveTemplate:   profilePickerActive,
		InactiveTemplate: profilePickerInactive,
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
	return picked, nil
}

// typedLocation returns a catalog and schema typed by the user in the form --location takes: trimmed and
// in lowercase.
func typedLocation(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// serviceRowChrome is the width the picker takes around a row's label: the border, the cursor and the checkbox.
const serviceRowChrome = 8

// markdownLink matches a [text](url) link, and bareURL a URL that is not in a link. They are compiled on first use, not
// in a package-level var, so that every databricks invocation does not pay for them.
var (
	markdownLink = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`) })
	bareURL      = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`https?://\S+`) })
)

// serviceDescription is the description of a service row: the CLI's Summary for a known system.ai service, else its
// Label, else the first sentence of the API comment.
func serviceDescription(s mcp.Service) string {
	if k := s.Known; k != nil {
		if k.Summary != "" {
			return k.Summary
		}
		if k.Label != "" {
			return k.Label
		}
	}
	text := markdownLink().ReplaceAllString(s.Comment, "$1")
	text = bareURL().ReplaceAllString(text, "")
	text = strings.Join(strings.Fields(text), " ")
	if i := strings.Index(text, ". "); i >= 0 {
		text = text[:i+1]
	}
	return text
}

// serviceChoiceLabel is a service picker row: the leaf name, its description and its tags, on one line of at most
// width columns (no limit when width is 0). The description is cut to make room for the tags, which always show.
func serviceChoiceLabel(s mcp.Service, installed bool, width int) string {
	var tags []string
	if k := s.Known; k != nil {
		if k.Recommended {
			tags = append(tags, "recommended")
		}
		if k.Beta {
			tags = append(tags, "beta")
		}
	}
	if s.SignInRequired() {
		tags = append(tags, "sign-in required")
	}
	var suffix string
	if len(tags) > 0 {
		suffix += "  (" + strings.Join(tags, ", ") + ")"
	}
	if installed {
		suffix += "  (installed)"
	}

	label := s.Leaf
	desc := serviceDescription(s)
	if desc == "" {
		return label + suffix
	}
	if width > 0 {
		room := width - ansi.StringWidth(label+suffix) - 2
		if room <= 0 {
			return label + suffix
		}
		desc = ansi.Truncate(desc, room, "…")
	}
	return label + "  " + desc + suffix
}

// serviceRowLabel is the label of a service picker row. A service the profile has installed is marked so.
func serviceRowLabel(r serviceRow, width int) string {
	return serviceChoiceLabel(r.service, r.installed, width)
}

// recommendedRank orders the rows of the service picker: 0 for a recommended service, else 1.
func recommendedRank(r serviceRow) int {
	if r.service.Known != nil && r.service.Known.Recommended {
		return 0
	}
	return 1
}

// multiSelectHelp is the help line of the multi-selects of mcp add and remove.
const multiSelectHelp = "space to toggle, enter to confirm"

// multiSelectSearchHelp is the help line of a multi-select while it searches: the keys type into the search, so a
// space is typed, and enter or esc leaves the search before space toggles a row.
const multiSelectSearchHelp = "type to search, enter or esc ends the search, then space toggles"

// multiSelectDescription is the help line of a multi-select, after the scope it picks in ("" for none). It names the
// search keys while the multi-select searches.
func multiSelectDescription(scope string, searching bool) string {
	help := multiSelectHelp
	if searching {
		help = multiSelectSearchHelp
	}
	if scope == "" {
		return help
	}
	return scope + ": " + help
}

// newMultiSelectPicker returns the picker of a multi-select in location ("" for none). Its help line follows the
// search.
func newMultiSelectPicker[T comparable](field *huh.MultiSelect[T], location string) *pickerModel {
	p := newPicker(huh.NewForm(huh.NewGroup(field)).WithShowHelp(false), field.GetFiltering)
	p.onKey = func() {
		field.Description(multiSelectDescription(location, field.GetFiltering()))
	}
	return p
}

// serviceSelectField is the multi-select over the rows of a schema, whose rows fit width columns when width is set.
// Its values go to selected.
func serviceSelectField(location string, rows []serviceRow, width int, selected *[]string) *huh.MultiSelect[string] {
	if width > 0 {
		width -= serviceRowChrome
	}
	// The recommended rows come first, so the picker opens at the top with the pre-checked ones in view. The API
	// order holds within each group.
	ordered := slices.Clone(rows)
	slices.SortStableFunc(ordered, func(a, b serviceRow) int {
		return cmp.Compare(recommendedRank(a), recommendedRank(b))
	})
	options := make([]huh.Option[string], 0, len(rows)+1)
	for _, r := range ordered {
		options = append(options, huh.NewOption(serviceRowLabel(r, width), r.service.FullName).Selected(r.checked))
	}
	options = append(options, huh.NewOption("Another schema...", anotherSchemaValue))

	return huh.NewMultiSelect[string]().
		Title("Select MCP servers").
		Description(multiSelectDescription(location, false)).
		Options(options...).
		Value(selected)
}

// terminalWidth is the width of the terminal the pickers draw in, or 0 when stderr is not a terminal, which leaves
// the rows whole.
func terminalWidth() int {
	width, _, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil {
		return 0
	}
	return width
}

// defaultPromptServiceSelection is the huh multi-select over the rows of a schema.
func defaultPromptServiceSelection(location string, rows []serviceRow) (servicePick, error) {
	var selected []string
	field := serviceSelectField(location, rows, terminalWidth(), &selected)
	if err := runPicker(newMultiSelectPicker(field, location)); err != nil {
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

// defaultPromptMCPAgents asks which agents get the servers. It lists the rows of install's agent picker, but with the keys
// of the other mcp pickers: Esc cancels it, and enter confirms the picks.
func defaultPromptMCPAgents(_ context.Context, scope string, choices []agentChoice) ([]*agents.Agent, error) {
	var selected []string
	field, byName := mcpAgentField(scope, choices, &selected)
	if err := runPicker(newMultiSelectPicker(field, "")); err != nil {
		return nil, promptError(err)
	}
	if len(selected) == 0 {
		return nil, errNoAgentsSelected
	}
	result := make([]*agents.Agent, 0, len(selected))
	for _, name := range selected {
		result = append(result, byName[name])
	}
	return result, nil
}

// mcpAgentField is the multi-select of the mcp agent picker over choices, whose values go to selected, and the agents by
// name.
func mcpAgentField(scope string, choices []agentChoice, selected *[]string) (*huh.MultiSelect[string], map[string]*agents.Agent) {
	options := make([]huh.Option[string], 0, len(choices))
	byName := make(map[string]*agents.Agent, len(choices))
	for _, c := range choices {
		options = append(options, huh.NewOption(c.label, c.agent.Name).Selected(c.preselect))
		byName[c.agent.Name] = c.agent
	}
	field := huh.NewMultiSelect[string]().
		Title(agentPickerTitle(scope)).
		Description(multiSelectHelp).
		Options(options...).
		Value(selected)
	return field, byName
}

// defaultPromptLocation asks for the catalog and schema to list next. An empty answer or Esc stops; see locationAnswer.
func defaultPromptLocation() (string, error) {
	var location string
	input := huh.NewInput().
		Title("Catalog and schema to list MCP servers from (leave empty or press esc to stop)").
		Value(&location).
		Validate(func(s string) error {
			if typed := typedLocation(s); typed != "" && validateLocation(typed) != nil {
				return errors.New("enter <catalog>.<schema>")
			}
			return nil
		})
	p := newPicker(huh.NewForm(huh.NewGroup(input)).WithShowHelp(false), nil)
	err := runPicker(p)
	return locationAnswer(p.cancelled, location, err)
}

// locationAnswer is the result of the location prompt. Esc stops listing, as an empty answer does, so that the picks of
// the pages already shown are kept; Ctrl+C still aborts the add.
func locationAnswer(cancelled bool, location string, err error) (string, error) {
	if cancelled {
		return "", nil
	}
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

// fitRow cuts a row label to the terminal width, so that each row stays on one line and the cursor stays in view. A
// width of 0 leaves the label whole.
func fitRow(label string, width int) string {
	if width <= 0 {
		return label
	}
	return ansi.Truncate(label, width-serviceRowChrome, "…")
}

// defaultPromptRemoveSelection is the huh multi-select over the installed servers.
func defaultPromptRemoveSelection(entries []mcp.Entry, targets []*agents.Agent) ([]mcp.Entry, error) {
	width := terminalWidth()
	options := make([]huh.Option[int], 0, len(entries))
	for i, e := range entries {
		options = append(options, huh.NewOption(fitRow(removeChoiceLabel(e, targets), width), i))
	}

	var selected []int
	field := huh.NewMultiSelect[int]().
		Title("Select MCP servers to remove").
		Description(multiSelectHelp).
		Options(options...).
		Value(&selected)
	if err := runPicker(newMultiSelectPicker(field, "")); err != nil {
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
	confirm := huh.NewConfirm().
		Title(service + " needs a one-time sign-in. Sign in now?").
		Value(&yes)
	if err := runPicker(newPicker(huh.NewForm(huh.NewGroup(confirm)).WithShowHelp(false), nil)); err != nil {
		return false, err
	}
	return yes, nil
}

// newProceedPicker returns the picker of the plan confirm, which defaults to yes and writes the answer to proceed.
func newProceedPicker(proceed *bool) *pickerModel {
	confirm := huh.NewConfirm().
		Title("Proceed?").
		Description("enter to confirm, esc to cancel").
		Value(proceed)
	return newPicker(huh.NewForm(huh.NewGroup(confirm)).WithShowHelp(false), nil)
}

// defaultPromptMCPProceed asks whether to carry out the plan of mcp add or mcp remove. The default is yes. Esc cancels
// as Ctrl+C does.
func defaultPromptMCPProceed() (bool, error) {
	proceed := true
	if err := runPicker(newProceedPicker(&proceed)); err != nil {
		return false, err
	}
	return proceed, nil
}
