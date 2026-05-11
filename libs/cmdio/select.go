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
)

const (
	// viewportSize is the number of list rows shown at once.
	viewportSize = 5

	gutterUp   = "↑ "
	gutterDown = "↓ "
	gutter     = "  "

	// Default templates match promptui's Select defaults: a blue "?" before
	// the label, a bold "▸" + underlined item for the active row, "  " +
	// item for inactive rows, and a green "✔" + faint item for the
	// post-submit summary. They render reasonably for any item type whose
	// fmt.Stringer / printf'd form is a single line.
	defaultLabelTemplate    = `{{ "?" | blue }} {{.}}: `
	defaultActiveTemplate   = `{{ "▸" | bold }} {{ . | underline }}`
	defaultInactiveTemplate = `  {{.}}`
	defaultSelectedTemplate = `{{ "✔" | green }} {{ . | faint }}`

	// helpTextBase is shown above the label when search isn't active and
	// HideHelp is unset; promptui renders its Help template entirely faint.
	// When a Searcher is configured the " and / toggles search" suffix is
	// appended to advertise the toggle.
	helpTextBase   = "Use the arrow keys to navigate: ↓ ↑ → ←"
	helpTextSearch = helpTextBase + "  and / toggles search"
)

// promptFuncMap is the pipeline-form template.FuncMap used by select-prompt
// templates (`{{ . | bold }}`). It always emits SGR codes; color.go's
// RenderFuncMap is printf-style and gates colors on ctx so the two cannot
// share an implementation.
var promptFuncMap = template.FuncMap{
	"bold":      promptANSI(ansiBold),
	"faint":     promptANSI(ansiFaint),
	"italic":    promptANSI(ansiItalic),
	"underline": promptANSI(ansiUnderline),
	"red":       promptANSI(ansiRed),
	"green":     promptANSI(ansiGreen),
	"yellow":    promptANSI(ansiYellow),
	"blue":      promptANSI(ansiBlue),
	"magenta":   promptANSI(ansiMagenta),
	"cyan":      promptANSI(ansiCyan),
}

func promptANSI(prefix string) func(any) string {
	return func(v any) string {
		return prefix + fmt.Sprint(v) + ansiReset
	}
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
	m, err := newSelectModel(opts)
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

func newSelectModel(opts SelectOptions) (*selectModel, error) {
	items, err := normalizeItems(opts.Items)
	if err != nil {
		return nil, err
	}
	labelTpl, err := parsePromptTemplate("label", defaultIfEmpty(opts.LabelTemplate, defaultLabelTemplate))
	if err != nil {
		return nil, err
	}
	activeTpl, err := parsePromptTemplate("active", defaultIfEmpty(opts.Active, defaultActiveTemplate))
	if err != nil {
		return nil, err
	}
	inactiveTpl, err := parsePromptTemplate("inactive", defaultIfEmpty(opts.Inactive, defaultInactiveTemplate))
	if err != nil {
		return nil, err
	}
	selectedTpl, err := parsePromptTemplate("selected", defaultIfEmpty(opts.Selected, defaultSelectedTemplate))
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

func parsePromptTemplate(name, src string) (*template.Template, error) {
	t, err := template.New(name).Funcs(promptFuncMap).Parse(src)
	if err != nil {
		return nil, fmt.Errorf("parse %s template: %w", name, err)
	}
	return t, nil
}

func (m *selectModel) Init() tea.Cmd { return tea.HideCursor }

func (m *selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tea.KeyCtrlC:
		m.cancelled = true
		return m, tea.Quit

	case tea.KeyEnter, tea.KeyCtrlJ:
		// Enter on an empty filtered list is intentionally inert; only
		// Ctrl+C escapes from a "No results" panel.
		if len(m.matches) == 0 {
			return m, nil
		}
		m.submitted = true
		return m, tea.Quit

	case tea.KeyUp, tea.KeyCtrlP:
		// Ctrl+P is readline's CharPrev; promptui's select listener handles
		// it identically to the up arrow.
		m.cursorUp()

	case tea.KeyDown, tea.KeyCtrlN:
		// Ctrl+N is readline's CharNext; same as the down arrow.
		m.cursorDown()

	case tea.KeyLeft, tea.KeyCtrlB:
		// Left arrow / Ctrl+B page up; promptui binds both to its
		// list.PageUp via KeyBackward.
		m.pageUp()

	case tea.KeyRight, tea.KeyCtrlF:
		// Right arrow / Ctrl+F page down; both map to KeyForward in
		// promptui and drive list.PageDown.
		m.pageDown()

	case tea.KeyBackspace, tea.KeyCtrlH:
		// Backspace sends DEL, Ctrl+H sends BS; chzyer/readline treats both
		// as CharBackspace inside the search buffer.
		//
		// Alt+Backspace is readline's word-delete combo; promptui drops it,
		// so we drop it here too instead of deleting one rune.
		if key.Alt {
			return m, nil
		}
		if !m.searchActive || m.filter == "" {
			return m, nil
		}
		if r, size := utf8.DecodeLastRuneInString(m.filter); r != utf8.RuneError {
			m.filter = m.filter[:len(m.filter)-size]
		}
		m.recomputeMatches()

	case tea.KeyTab, tea.KeyRunes, tea.KeySpace:
		// Alt+<rune> are readline word-nav combos promptui ignores; don't
		// let them sneak in as filter input.
		if key.Alt {
			return m, nil
		}
		if !m.searchActive {
			// Outside search mode, vim-style shortcuts navigate the list and
			// "/" toggles search when a Searcher is configured. Anything
			// else is dropped. Multiple runes can arrive in a single KeyMsg
			// when the user types quickly, so dispatch per-rune.
			if key.Type == tea.KeyRunes {
				for _, r := range key.Runes {
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
			}
			return m, nil
		}
		switch key.Type {
		case tea.KeyTab:
			m.filter += "\t"
		case tea.KeySpace:
			m.filter += " "
		default:
			// "/" toggles search off (matching promptui): clear the filter
			// and exit search mode. Any other runes in this KeyMsg are
			// dropped — promptui dispatches per-rune so this only matters
			// when bubbletea batches keystrokes, in which case the user
			// almost certainly meant the toggle.
			if slices.Contains(key.Runes, '/') {
				m.filter = ""
				m.searchActive = false
				m.recomputeMatches()
				return m, nil
			}
			m.filter += string(key.Runes)
		}
		m.recomputeMatches()

	case tea.KeyEsc:
		// Esc is intentionally inert.

	default:
		// Other key types (Home/End, Ctrl+U/W, function keys, …) are no-ops.
	}
	return m, nil
}

func (m *selectModel) View() string {
	var b strings.Builder

	// After submission render only the Selected template — promptui replaced
	// the prompt UI with a single-line confirmation, and the post-quit frame
	// stays on screen as the user-visible result. HideSelected callers leave
	// the screen blank.
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
// new top if it was below it. Mirrors promptui's list.PageUp.
func (m *selectModel) pageUp() {
	m.viewportTop = max(m.viewportTop-viewportSize, 0)
	if m.viewportTop < m.cursor {
		m.cursor = m.viewportTop
	}
}

// pageDown shifts the viewport down by one page, clamping so the last full
// page stays visible, then bumps the cursor up to the new top if it lagged
// behind. Mirrors promptui's list.PageDown.
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
		return 0, errCtrlC
	}
	return sm.originalIndex(), nil
}
