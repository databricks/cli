package aircmd

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// DuBois dark-first palette. Neutrals (n7..n12) carry structure; the state
// colors are reserved for the Status column.
const (
	colN7      = lipgloss.Color("#69696B") // gutter / dim
	colN9      = lipgloss.Color("#ABABAE") // secondary values
	colN11     = lipgloss.Color("#E0E0E3") // body
	colN12     = lipgloss.Color("#F1F1F4") // selected text
	colOverlay = lipgloss.Color("#1F1F23") // selected-row fill
	colRunID   = lipgloss.Color("#B7A8E8") // run id
	colGreen   = lipgloss.Color("#4CD964") // success
	colAmber   = lipgloss.Color("#E8B84A") // running / pending
	colRed     = lipgloss.Color("#EB6B6B") // failed
	colBlue    = lipgloss.Color("#6CA8F0") // MLflow link
)

const mlflowColWidth = 22

// listStyles renders the runs table. The renderer carries the color profile, so
// styles render plain under --no-color / non-tty.
type listStyles struct {
	r *lipgloss.Renderer
}

func newListStyles(r *lipgloss.Renderer) listStyles {
	return listStyles{r: r}
}

// listColumn identifies a table column, in display order.
type listColumn int

const (
	columnRunID listColumn = iota
	columnExperiment
	columnStatus
	columnStarted
	columnDuration
	columnProgress
	columnMLflow
	columnUser
	columnAccel
)

// numListColumns is how many columns the table has.
const numListColumns = int(columnAccel) + 1

// listColumnNames are the header titles, indexed by listColumn.
var listColumnNames = [numListColumns]string{
	"Run ID", "Experiment", "Status", "Started", "Duration", "Progress", "MLflow", "User", "Accelerators",
}

// hideOrder lists the columns a too-narrow terminal drops, lowest priority
// first.
var hideOrder = []listColumn{columnAccel, columnUser, columnMLflow, columnProgress}

// listCols holds the computed width of each column; the gutter is one cell.
// compactStarted switches Started to its short form; hidden columns are left
// out of the rendered rows.
type listCols struct {
	runID, experiment, status, started, duration, progress, mlflow, user, accel int
	compactStarted                                                              bool
	hidden                                                                      uint16 // bit i set when listColumn i is hidden
}

// columnCap bounds the widest free-text columns so one long value can't dominate
// the row.
const columnCap = 36

// computeListCols sizes every column to its content: the layout of the one-shot
// table, and the starting point for fitting the interactive one.
func computeListCols(rows []listRow) listCols {
	c := listCols{
		runID:      len("Run ID"),
		experiment: len("Experiment"),
		status:     len("Status"),
		started:    len("Started"),
		duration:   len("Duration"),
		progress:   len("Progress"),
		mlflow:     mlflowColWidth,
		user:       len("User"),
		accel:      len("Accelerators"),
	}
	for _, r := range rows {
		c.runID = max(c.runID, lipgloss.Width(r.RunID))
		c.experiment = min(columnCap, max(c.experiment, lipgloss.Width(r.Experiment)))
		c.status = max(c.status, lipgloss.Width("● "+listRowStatus(r)))
		c.started = max(c.started, lipgloss.Width(startedDisplay(r)))
		c.duration = max(c.duration, lipgloss.Width(r.Duration))
		c.progress = max(c.progress, lipgloss.Width(progressDisplay(r)))
		c.user = min(columnCap, max(c.user, lipgloss.Width(r.User)))
		c.accel = max(c.accel, lipgloss.Width(r.Accelerators))
	}
	return c
}

// Floors for the flexible columns: narrow enough to give room back to the fixed
// columns, wide enough that a truncated value is still recognizable.
const (
	experimentMinWidth = 12
	userMinWidth       = 8
)

// width is the cell width of col.
func (c listCols) width(col listColumn) int {
	switch col {
	case columnRunID:
		return c.runID
	case columnExperiment:
		return c.experiment
	case columnStatus:
		return c.status
	case columnStarted:
		return c.started
	case columnDuration:
		return c.duration
	case columnProgress:
		return c.progress
	case columnMLflow:
		return c.mlflow
	case columnUser:
		return c.user
	case columnAccel:
		return c.accel
	}
	return 0
}

// shown reports whether col is part of the rendered table.
func (c listCols) shown(col listColumn) bool {
	return c.hidden&(1<<col) == 0
}

// hiddenNames lists the hidden columns' titles in table order; nil when every
// column is shown.
func (c listCols) hiddenNames() []string {
	var names []string
	for col := range listColumn(numListColumns) {
		if !c.shown(col) {
			names = append(names, listColumnNames[col])
		}
	}
	return names
}

// tableWidth is the full width of a rendered row: the gutter plus every shown
// column and its two-cell separator.
func (c listCols) tableWidth() int {
	w := 1
	for col := range listColumn(numListColumns) {
		if c.shown(col) {
			w += c.width(col) + 2
		}
	}
	return w
}

// fitWidth shrinks the flexible columns (Experiment, User) until the table fits
// in width cells, so a long value truncates instead of pushing the columns to
// its right off screen. It takes from the wider column first and stops at the
// floors: a terminal too narrow for the fixed columns still overflows. A width
// of 0 means the terminal size is unknown and leaves the columns alone.
func (c listCols) fitWidth(width int) listCols {
	if width <= 0 {
		return c
	}
	experimentFloor := min(c.experiment, experimentMinWidth)
	userFloor := min(c.user, userMinWidth)
	for overflow := c.tableWidth() - width; overflow > 0; overflow-- {
		canShrinkExperiment := c.experiment > experimentFloor
		canShrinkUser := c.shown(columnUser) && c.user > userFloor
		switch {
		case canShrinkExperiment && (!canShrinkUser || c.experiment >= c.user):
			c.experiment--
		case canShrinkUser:
			c.user--
		default:
			return c
		}
	}
	return c
}

// compact switches the fixed columns to their short forms: Started as
// "MM-DD HH:MM" and MLflow as wide as its widest label rather than the full
// mlflowColWidth. Labels were already cut to that width when the rows were
// built, so they are never shortened again here.
func (c listCols) compact(rows []listRow) listCols {
	c.compactStarted = true
	c.started = len("Started")
	c.mlflow = len("MLflow")
	for _, r := range rows {
		c.started = max(c.started, lipgloss.Width(startedCompact(r)))
		c.mlflow = max(c.mlflow, lipgloss.Width(mlflowLabel(r)))
	}
	return c
}

// layoutListCols lays the interactive table out for a terminal width cells
// wide. Experiment and User shrink to their floors first; if the table is still
// too wide the fixed columns compact, and then whole columns are hidden in
// hideOrder until it fits. Each step gives its spare cells back to Experiment
// and User. A terminal that fits today's table gets today's layout, and a width
// of 0 (unknown) leaves the natural widths alone.
func layoutListCols(rows []listRow, width int) listCols {
	natural := computeListCols(rows)
	if width <= 0 {
		return natural
	}
	cols := natural.fitWidth(width)
	if cols.tableWidth() <= width {
		return cols
	}
	compact := natural.compact(rows)
	cols = compact.fitWidth(width)
	for _, col := range hideOrder {
		if cols.tableWidth() <= width {
			break
		}
		compact.hidden |= 1 << col
		cols = compact.fitWidth(width)
	}
	return cols
}

// renderHeader renders the muted column-title row.
func (s listStyles) renderHeader(cols listCols) string {
	h := func(text string, width int, right bool) string {
		return s.r.NewStyle().Foreground(colN9).Render(pad(text, width, right))
	}
	cells := []string{pad(" ", 1, false)}
	for col := range listColumn(numListColumns) {
		if cols.shown(col) {
			cells = append(cells, h(listColumnNames[col], cols.width(col), col == columnDuration || col == columnProgress))
		}
	}
	// Trim trailing pad on the final column so rows carry no trailing whitespace.
	return strings.TrimRight(strings.Join(cells, "  "), " ")
}

// renderRow renders one run. The selected row uses a subtle overlay fill + n12
// text (not full inversion, not per-state color).
func (s listStyles) renderRow(cols listCols, r listRow, selected, links bool) string {
	base := s.r.NewStyle()
	if selected {
		base = base.Background(colOverlay)
	}
	fg := func(c lipgloss.Color) lipgloss.Color {
		if selected {
			return colN12
		}
		return c
	}

	gutter := " "
	if selected {
		gutter = "▸"
	}

	runIDLink := ""
	if links {
		runIDLink = r.RunURL
	}
	experimentLink := ""
	if links {
		experimentLink = r.ExperimentURL
	}

	// Underline only cells that actually carry a link, so unlinked text isn't
	// styled as clickable.
	cell := func(col listColumn) string {
		switch col {
		case columnRunID:
			return s.cell(base, r.RunID, cols.runID, fg(colRunID), false, runIDLink != "", runIDLink)
		case columnExperiment:
			return s.cell(base, r.Experiment, cols.experiment, fg(colN11), false, experimentLink != "", experimentLink)
		case columnStatus:
			return s.cell(base, "● "+listRowStatus(r), cols.status, fg(statusColor(listRowStatus(r))), false, false, "")
		case columnStarted:
			return s.cell(base, cols.startedText(r), cols.started, fg(colN9), false, false, "")
		case columnDuration:
			return s.cell(base, r.Duration, cols.duration, fg(colN9), true, false, "")
		case columnProgress:
			return s.cell(base, progressDisplay(r), cols.progress, fg(colAmber), true, false, "")
		case columnMLflow:
			return s.mlflowCell(base, cols.mlflow, r, selected, links)
		case columnUser:
			return s.cell(base, r.User, cols.user, fg(colN9), false, false, "")
		case columnAccel:
			return s.cell(base, r.Accelerators, cols.accel, fg(colN9), false, false, "")
		}
		return ""
	}
	cells := []string{s.cell(base, gutter, 1, fg(colN7), false, false, "")}
	for col := range listColumn(numListColumns) {
		if cols.shown(col) {
			cells = append(cells, cell(col))
		}
	}
	// Trim the final column's trailing pad so rows carry no trailing whitespace.
	return strings.TrimRight(strings.Join(cells, base.Render("  ")), " ")
}

func listRowStatus(r listRow) string {
	if r.DisplayStatus != "" {
		return r.DisplayStatus
	}
	return r.Status
}

// cell renders one padded, colored cell. The text is truncated to width, then
// padded with (background-only) spaces so columns align even when the text is
// styled or hyperlinked.
func (s listStyles) cell(base lipgloss.Style, text string, width int, fg lipgloss.Color, right, underline bool, link string) string {
	text = truncate(text, width)
	style := base.Foreground(fg)
	if underline {
		style = style.Underline(true)
	}
	rendered := style.Render(text)
	if link != "" {
		rendered = termenv.Hyperlink(link, rendered)
	}
	gap := max(width-lipgloss.Width(text), 0)
	padStr := base.Render(strings.Repeat(" ", gap))
	if right {
		return padStr + rendered
	}
	return rendered + padStr
}

// mlflowCell renders the MLflow column: a short, blue, underlined OSC 8
// hyperlink (when links are enabled), or "-" when the run has no link.
func (s listStyles) mlflowCell(base lipgloss.Style, width int, r listRow, selected, links bool) string {
	if r.MLflowURL == "" || r.MLflowURL == "-" {
		fg := colN9
		if selected {
			fg = colN12
		}
		return s.cell(base, "-", width, fg, false, false, "")
	}
	fg := colBlue
	if selected {
		fg = colN12
	}
	link := ""
	if links {
		link = r.MLflowURL
	}
	return s.cell(base, mlflowLabel(r), width, fg, false, true, link)
}

// mlflowLabel is the text shown in the row's MLflow cell: its label, or "-"
// when the run has no link or no label.
func mlflowLabel(r listRow) string {
	if r.MLflowURL == "" || r.MLflowURL == "-" || r.MLflowLabel == "" || r.MLflowLabel == "-" {
		return "-"
	}
	return r.MLflowLabel
}

// statusColor maps an air run status word to its data color.
func statusColor(status string) lipgloss.Color {
	switch status {
	case "SUCCESS":
		return colGreen
	case "RUNNING", "PENDING", "TERMINATING":
		return colAmber
	case "FAILED":
		return colRed
	default: // CANCELED / UNKNOWN
		return colN7
	}
}

// progressDisplay is the row's progress estimate, or "-" when there is none
// (a finished run, or a run we couldn't estimate).
func progressDisplay(r listRow) string {
	if r.Progress == "" {
		return "-"
	}
	return r.Progress
}

// startedDisplay trims the row's ISO start timestamp to second precision
// ("2006-01-02T15:04:05"), or "-" when the run hasn't started.
func startedDisplay(r listRow) string {
	if r.StartedAt == nil {
		return "-"
	}
	s := *r.StartedAt
	if len(s) >= 19 {
		return s[:19]
	}
	return s
}

// startedCompact is the short form of the row's start timestamp for a narrow
// terminal ("01-02 15:04"), or "-" when the run hasn't started. A timestamp too
// short to carry a date and time is shown as is.
func startedCompact(r listRow) string {
	if r.StartedAt == nil {
		return "-"
	}
	s := *r.StartedAt
	if len(s) >= 16 {
		return s[5:10] + " " + s[11:16]
	}
	return s
}

// startedText is the row's Started cell in the layout's long or compact form.
func (c listCols) startedText(r listRow) string {
	if c.compactStarted {
		return startedCompact(r)
	}
	return startedDisplay(r)
}

// pad pads (or truncates) s to a visible width of n, right-aligned when right is
// set. It measures visible width, so it is safe on styled strings.
func pad(s string, n int, right bool) string {
	s = truncate(s, n)
	gap := max(n-lipgloss.Width(s), 0)
	if gap == 0 {
		return s
	}
	fill := strings.Repeat(" ", gap)
	if right {
		return fill + s
	}
	return s + fill
}

// truncate shortens s to a visible width of n, appending "…" on overflow.
func truncate(s string, n int) string {
	return ansi.Truncate(s, n, "…")
}

// truncateStart shortens s to a visible width of n, keeping its end behind a
// leading "…" on overflow.
func truncateStart(s string, n int) string {
	width := lipgloss.Width(s)
	if width <= n {
		return s
	}
	if n <= 1 {
		return truncate(s, n)
	}
	// TruncateLeft never splits a character, so a cut that lands inside a
	// two-cell character leaves the result one cell too wide.
	cut := width - (n - 1)
	out := ansi.TruncateLeft(s, cut, "…")
	if lipgloss.Width(out) > n {
		out = ansi.TruncateLeft(s, cut+1, "…")
	}
	// TruncateLeft drops the "…" when nothing of s is left.
	if out == "" {
		return "…"
	}
	return out
}
