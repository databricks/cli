package cmdio

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"text/template"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/databricks/cli/libs/cmdio/teainput"
)

const (
	// viewportSize is the number of list rows shown at once.
	viewportSize = 5

	gutterUp   = "↑ "
	gutterDown = "↓ "
	gutter     = "  "

	// Default templates: blue "?" before the label, bold "▸" + underlined
	// item for the active row, two-space prefix + item for inactive rows,
	// green "✔" + faint item for the post-submit summary. They render
	// reasonably for any item type whose fmt.Stringer / printf'd form is
	// a single line.
	defaultLabelTemplate    = `{{ "?" | blue }} {{.}}: `
	defaultActiveTemplate   = `{{ "▸" | bold }} {{ . | underline }}`
	defaultInactiveTemplate = `  {{.}}`
	defaultSelectedTemplate = `{{ "✔" | green }} {{ . | faint }}`

	// helpTextBase is shown faint above the label when search isn't active
	// and HideHelp is unset. When a Searcher is configured the trailing
	// suffix advertises the "/" toggle.
	helpTextBase   = "Use the arrow keys to navigate: ↓ ↑ → ←"
	helpTextSearch = helpTextBase + "  and / toggles search"
)

// promptFuncMap returns the pipeline-form template.FuncMap used by
// select-prompt templates (`{{ . | bold }}`). color.go's RenderFuncMap is
// printf-style, so the two cannot share an implementation. With color off
// (NO_COLOR) the color helpers pass text through unchanged; bold, faint,
// italic and underline are not colors and always apply.
func promptFuncMap(color bool) template.FuncMap {
	return template.FuncMap{
		"bold":      promptANSI(ansiBold),
		"faint":     promptANSI(ansiFaint),
		"italic":    promptANSI(ansiItalic),
		"underline": promptANSI(ansiUnderline),
		"red":       promptANSI(colorCode(color, ansiRed)),
		"green":     promptANSI(colorCode(color, ansiGreen)),
		"yellow":    promptANSI(colorCode(color, ansiYellow)),
		"blue":      promptANSI(colorCode(color, ansiBlue)),
		"magenta":   promptANSI(colorCode(color, ansiMagenta)),
		"cyan":      promptANSI(colorCode(color, ansiCyan)),
	}
}

// promptANSI wraps a value in an SGR code. An empty code (a color with color
// off) passes the value through unchanged.
func promptANSI(prefix string) func(any) string {
	if prefix == "" {
		return func(v any) string { return fmt.Sprint(v) }
	}
	return func(v any) string {
		return prefix + fmt.Sprint(v) + ansiReset
	}
}

// colorCode returns code, or "" with color off. Prompts use it for colors
// only: bold, faint, italic and underline are not colors, so NO_COLOR keeps
// them.
func colorCode(color bool, code string) string {
	if !color {
		return ""
	}
	return code
}

// SelectOptions configures an interactive single-choice picker shown by
// [RunSelect]. Template strings use text/template syntax and have access
// to the fields of the items in Items.
type SelectOptions struct {
	// Label is shown above the list. Required.
	Label string

	// Items is the slice of values to choose from. Templates reference
	// fields on the element type.
	Items any

	// Searcher, when set, narrows the list as the user types.
	Searcher func(input string, index int) bool

	// StartInSearchMode opens the prompt with the search input focused.
	StartInSearchMode bool

	// HideHelp hides the navigation help line shown by default when no
	// search is active.
	HideHelp bool

	// HideSelected hides the rendered selection after the prompt closes.
	HideSelected bool

	// LabelTemplate renders Label. Empty uses the default.
	LabelTemplate string

	// Active renders the highlighted item.
	Active string

	// Inactive renders non-highlighted items.
	Inactive string

	// Selected renders the chosen item after the prompt closes.
	Selected string
}

// RunSelect shows an interactive picker and returns the index of the chosen
// item. Returns an error without prompting when the terminal does not support
// it.
func RunSelect(ctx context.Context, opts SelectOptions) (int, error) {
	c := fromContext(ctx)
	if !c.capabilities.SupportsPrompt() {
		return 0, fmt.Errorf("expected to have %s", opts.Label)
	}
	m, err := newSelectModel(opts, c.capabilities.SupportsStderrColor())
	if err != nil {
		return 0, err
	}
	return c.runSelectModel(m)
}

// Tuple pairs a human-friendly Name with an internal Id, used as the row type
// for [Select] and [SelectOrdered].
type Tuple struct{ Name, Id string }

// Select shows a selection prompt where the user can pick one of the name/id
// items. The items are sorted alphabetically by name.
func Select[V any](ctx context.Context, names map[string]V, label string) (string, error) {
	items := make([]Tuple, 0, len(names))
	for k, v := range names {
		items = append(items, Tuple{k, fmt.Sprint(v)})
	}
	slices.SortFunc(items, func(a, b Tuple) int {
		return strings.Compare(a.Name, b.Name)
	})
	return SelectOrdered(ctx, items, label)
}

// SelectOrdered shows a selection prompt where the user can pick one of the
// name/id items. The items appear in the order specified in the "items"
// argument.
func SelectOrdered(ctx context.Context, items []Tuple, label string) (string, error) {
	idx, err := RunSelect(ctx, SelectOptions{
		Label:             label,
		Items:             items,
		HideSelected:      true,
		StartInSearchMode: true,
		Searcher: func(input string, idx int) bool {
			return strings.Contains(strings.ToLower(items[idx].Name), strings.ToLower(input))
		},
		Active:   `{{.Name | bold}} ({{.Id|faint}})`,
		Inactive: `{{.Name}}`,
	})
	if err != nil {
		return "", err
	}
	return items[idx].Id, nil
}

type selectModel struct {
	label string
	items []any

	labelTpl    *template.Template
	activeTpl   *template.Template
	inactiveTpl *template.Template
	selectedTpl *template.Template

	searcher     func(input string, idx int) bool
	searchActive bool
	hideHelp     bool
	hideSelected bool

	filter      string
	matches     []int
	cursor      int
	viewportTop int

	submitted bool
	cancelled bool
}

func newSelectModel(opts SelectOptions, color bool) (*selectModel, error) {
	items, err := normalizeItems(opts.Items)
	if err != nil {
		return nil, err
	}
	funcs := promptFuncMap(color)
	labelTpl, err := parsePromptTemplate(funcs, "label", defaultIfEmpty(opts.LabelTemplate, defaultLabelTemplate))
	if err != nil {
		return nil, err
	}
	activeTpl, err := parsePromptTemplate(funcs, "active", defaultIfEmpty(opts.Active, defaultActiveTemplate))
	if err != nil {
		return nil, err
	}
	inactiveTpl, err := parsePromptTemplate(funcs, "inactive", defaultIfEmpty(opts.Inactive, defaultInactiveTemplate))
	if err != nil {
		return nil, err
	}
	selectedTpl, err := parsePromptTemplate(funcs, "selected", defaultIfEmpty(opts.Selected, defaultSelectedTemplate))
	if err != nil {
		return nil, err
	}

	m := &selectModel{
		label:        opts.Label,
		items:        items,
		labelTpl:     labelTpl,
		activeTpl:    activeTpl,
		inactiveTpl:  inactiveTpl,
		selectedTpl:  selectedTpl,
		searcher:     opts.Searcher,
		searchActive: opts.StartInSearchMode,
		hideHelp:     opts.HideHelp,
		hideSelected: opts.HideSelected,
	}
	m.recomputeMatches()
	return m, nil
}

func defaultIfEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func parsePromptTemplate(funcs template.FuncMap, name, src string) (*template.Template, error) {
	t, err := template.New(name).Funcs(funcs).Parse(src)
	if err != nil {
		return nil, fmt.Errorf("parse %s template: %w", name, err)
	}
	return t, nil
}

func (m *selectModel) Init() tea.Cmd { return tea.HideCursor }

func (m *selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	in, ok := teainput.Decode(msg)
	if !ok {
		return m, nil
	}
	if in.Paste {
		// A paste is literal filter text: it doesn't navigate or toggle
		// search the way typed "j", "k" or "/" do.
		if m.searchActive && in.Text != "" {
			m.appendFilter(teainput.FirstLine(in.Text))
		}
		return m, nil
	}

	switch in.Key {
	case "ctrl+c":
		m.cancelled = true
		return m, tea.Quit

	case "enter", "ctrl+j":
		// Enter on an empty filtered list is intentionally inert; only
		// Ctrl+C escapes from a "No results" panel.
		if len(m.matches) == 0 {
			return m, nil
		}
		m.submitted = true
		return m, tea.Quit

	case "up", "ctrl+p":
		m.cursorUp()

	case "down", "ctrl+n":
		m.cursorDown()

	case "left", "ctrl+b":
		m.pageUp()

	case "right", "ctrl+f":
		m.pageDown()

	case "backspace", "ctrl+h":
		// Backspace sends DEL, Ctrl+H sends BS; treat both as filter
		// backspace. Alt+Backspace (word-delete) is named "alt+backspace"
		// and falls to the default branch rather than deleting a rune.
		if !m.searchActive || m.filter == "" {
			return m, nil
		}
		if r, size := utf8.DecodeLastRuneInString(m.filter); r != utf8.RuneError {
			m.filter = m.filter[:len(m.filter)-size]
		}
		m.recomputeMatches()

	case "tab":
		if m.searchActive {
			m.appendFilter("\t")
		}

	case "esc":
		// Esc is intentionally inert.

	default:
		// Alt+<rune> are word-nav combos and carry no Text, so they can't
		// sneak into the filter buffer. The baseline tests pin this as a
		// no-op for parity with the previous prompt library. Other keys
		// (Home/End, Ctrl+U/W, function keys, …) carry no Text either.
		if in.Text == "" {
			return m, nil
		}
		if !m.searchActive {
			// Outside search mode, vim-style shortcuts navigate the list and
			// "/" toggles search when a Searcher is configured. Anything
			// else is dropped. Multiple runes can arrive in a single event
			// when the user types quickly, so dispatch per-rune.
			for _, r := range in.Text {
				switch r {
				case 'j':
					m.cursorDown()
				case 'k':
					m.cursorUp()
				case 'l':
					m.pageDown()
				case 'h':
					m.pageUp()
				case '/':
					if m.searcher != nil {
						m.searchActive = true
					}
				}
			}
			return m, nil
		}
		// "/" toggles search off: clear the filter and exit search mode.
		// Any other runes batched in the same event are dropped — when the
		// user types "/" the surrounding characters almost certainly belong
		// to a toggle, not the filter.
		if strings.ContainsRune(in.Text, '/') {
			m.filter = ""
			m.searchActive = false
			m.recomputeMatches()
			return m, nil
		}
		m.appendFilter(in.Text)
	}
	return m, nil
}

func (m *selectModel) appendFilter(s string) {
	m.filter += s
	m.recomputeMatches()
}

func (m *selectModel) View() string { return m.render() }

func (m *selectModel) render() string {
	var b strings.Builder

	// After submission render only the Selected template; the post-quit
	// frame stays on screen as the user-visible result. HideSelected
	// callers leave the screen blank.
	//
	// The trailing "\n" is load-bearing for the same reason as in
	// promptModel.View: bubbletea's renderer wipes the cursor's row on
	// shutdown, so we park the cursor on an empty row below the content.
	if m.submitted {
		if !m.hideSelected {
			if err := m.selectedTpl.Execute(&b, m.items[m.originalIndex()]); err != nil {
				fmt.Fprintf(&b, "[selected template error: %v]", err)
			}
			b.WriteString("\n")
		}
		return b.String()
	}

	switch {
	case m.searchActive:
		b.WriteString("Search: ")
		// Tab stops every 8 columns from col 8 (after "Search: "). Expand to
		// spaces because tea's diff-based redraw doesn't reliably clear the
		// column the previous cursor occupied when a literal \t lands there,
		// leaving a stale █ behind.
		b.WriteString(expandTabsFromCol(m.filter, 8))
		b.WriteString(cursorBlock)
		b.WriteString("\n")
	case !m.hideHelp:
		text := helpTextBase
		if m.searcher != nil {
			text = helpTextSearch
		}
		b.WriteString(ansiFaint + text + ansiReset)
		b.WriteString("\n")
	}

	if err := m.labelTpl.Execute(&b, m.label); err != nil {
		fmt.Fprintf(&b, "[label template error: %v]", err)
	}
	b.WriteString("\n")

	if len(m.matches) == 0 {
		b.WriteString("\nNo results\n")
		return b.String()
	}

	end := min(m.viewportTop+viewportSize, len(m.matches))
	hasAbove := m.viewportTop > 0
	hasBelow := end < len(m.matches)

	for i := m.viewportTop; i < end; i++ {
		switch {
		case i == m.viewportTop && hasAbove:
			b.WriteString(gutterUp)
		case i == end-1 && hasBelow:
			b.WriteString(gutterDown)
		default:
			b.WriteString(gutter)
		}

		tpl := m.inactiveTpl
		if i == m.cursor {
			tpl = m.activeTpl
		}
		if err := tpl.Execute(&b, m.items[m.matches[i]]); err != nil {
			fmt.Fprintf(&b, "[template error: %v]", err)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *selectModel) recomputeMatches() {
	m.matches = m.matches[:0]
	for i := range m.items {
		if m.filter == "" || m.searcher == nil || m.searcher(m.filter, i) {
			m.matches = append(m.matches, i)
		}
	}
	m.cursor = 0
	m.viewportTop = 0
}

func (m *selectModel) cursorUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.viewportTop {
			m.viewportTop = m.cursor
		}
	}
}

func (m *selectModel) cursorDown() {
	if m.cursor < len(m.matches)-1 {
		m.cursor++
		if m.cursor >= m.viewportTop+viewportSize {
			m.viewportTop = m.cursor - viewportSize + 1
		}
	}
}

// pageUp shifts the viewport up by one page, then drops the cursor onto the
// new top if it was below it.
func (m *selectModel) pageUp() {
	m.viewportTop = max(m.viewportTop-viewportSize, 0)
	if m.viewportTop < m.cursor {
		m.cursor = m.viewportTop
	}
}

// pageDown shifts the viewport down by one page, clamping so the last full
// page stays visible, then bumps the cursor up to the new top if it lagged
// behind.
func (m *selectModel) pageDown() {
	max := len(m.matches) - viewportSize
	switch newTop := m.viewportTop + viewportSize; {
	case len(m.matches) < viewportSize:
		m.viewportTop = 0
	case newTop > max:
		m.viewportTop = max
	default:
		m.viewportTop = newTop
	}
	if m.cursor < m.viewportTop {
		m.cursor = m.viewportTop
	}
}

// originalIndex returns the items-slice index of the currently selected match,
// or -1 when nothing is selectable.
func (m *selectModel) originalIndex() int {
	if len(m.matches) == 0 {
		return -1
	}
	return m.matches[m.cursor]
}

// normalizeItems accepts any slice via reflection and copies its elements
// into a []any so the model can index them without further reflection. A
// non-slice argument is rejected at construction time.
func normalizeItems(in any) ([]any, error) {
	if in == nil {
		return nil, nil
	}
	v := reflect.ValueOf(in)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("SelectOptions.Items must be a slice, got %s", v.Kind())
	}
	out := make([]any, v.Len())
	for i := range out {
		out[i] = v.Index(i).Interface()
	}
	return out, nil
}

// expandTabsFromCol replaces \t in s with spaces, advancing to the next tab
// stop (every 8 columns) given a starting column.
func expandTabsFromCol(s string, startCol int) string {
	var b strings.Builder
	col := startCol
	for _, r := range s {
		if r == '\t' {
			stop := ((col / 8) + 1) * 8
			for col < stop {
				b.WriteByte(' ')
				col++
			}
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

func (c *cmdIO) runSelectModel(m *selectModel) (int, error) {
	final, err := c.runTUI(m)
	if err != nil {
		return 0, err
	}
	sm := final.(*selectModel)
	if sm.cancelled {
		return 0, ErrCtrlC
	}
	return sm.originalIndex(), nil
}
