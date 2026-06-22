package aircmd

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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

const mlflowColWidth = 18

// listStyles renders the runs table. The renderer carries the color profile, so
// styles render plain under --no-color / non-tty.
type listStyles struct {
	r *lipgloss.Renderer
}

func newListStyles(r *lipgloss.Renderer) listStyles {
	return listStyles{r: r}
}

// listCols holds the computed width of each variable-width column. MLflow is
// fixed (a short link) and the gutter is one cell.
type listCols struct {
	runID, experiment, status, started, duration, user, accel int
}

// columnCap bounds the widest free-text columns so one long value can't dominate
// the row.
const columnCap = 36

func computeListCols(rows []listRow) listCols {
	c := listCols{
		runID:      len("Run ID"),
		experiment: len("Experiment"),
		status:     len("Status"),
		started:    len("Started"),
		duration:   len("Duration"),
		user:       len("User"),
		accel:      len("Accelerators"),
	}
	for _, r := range rows {
		c.runID = max(c.runID, lipgloss.Width(r.RunID))
		c.experiment = min(columnCap, max(c.experiment, lipgloss.Width(r.Experiment)))
		c.status = max(c.status, lipgloss.Width("● "+r.Status))
		c.started = max(c.started, lipgloss.Width(startedDisplay(r)))
		c.duration = max(c.duration, lipgloss.Width(r.Duration))
		c.user = min(columnCap, max(c.user, lipgloss.Width(r.User)))
		c.accel = max(c.accel, lipgloss.Width(r.Accelerators))
	}
	return c
}

// renderHeader renders the muted column-title row.
func (s listStyles) renderHeader(cols listCols) string {
	h := func(text string, width int, right bool) string {
		return s.r.NewStyle().Foreground(colN9).Render(pad(text, width, right))
	}
	cells := []string{
		pad(" ", 1, false),
		h("Run ID", cols.runID, false),
		h("Experiment", cols.experiment, false),
		h("Status", cols.status, false),
		h("Started", cols.started, false),
		h("Duration", cols.duration, true),
		h("MLflow", mlflowColWidth, false),
		h("User", cols.user, false),
		h("Accelerators", cols.accel, false),
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

	cells := []string{
		s.cell(base, gutter, 1, fg(colN7), false, false, ""),
		s.cell(base, r.RunID, cols.runID, fg(colRunID), false, false, ""),
		s.cell(base, r.Experiment, cols.experiment, fg(colN11), false, false, ""),
		s.cell(base, "● "+r.Status, cols.status, fg(statusColor(r.Status)), false, false, ""),
		s.cell(base, startedDisplay(r), cols.started, fg(colN9), false, false, ""),
		s.cell(base, r.Duration, cols.duration, fg(colN9), true, false, ""),
		s.mlflowCell(base, r, selected, links),
		s.cell(base, r.User, cols.user, fg(colN9), false, false, ""),
		s.cell(base, r.Accelerators, cols.accel, fg(colN9), false, false, ""),
	}
	// Trim the final column's trailing pad so rows carry no trailing whitespace.
	return strings.TrimRight(strings.Join(cells, base.Render("  ")), " ")
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

// mlflowCell renders the fixed-width MLflow column: a short, blue, underlined
// OSC 8 hyperlink (when links are enabled), or "-" when the run has no link.
func (s listStyles) mlflowCell(base lipgloss.Style, r listRow, selected, links bool) string {
	if r.MLflowURL == "" || r.MLflowURL == "-" {
		fg := colN9
		if selected {
			fg = colN12
		}
		return s.cell(base, "-", mlflowColWidth, fg, false, false, "")
	}
	fg := colBlue
	if selected {
		fg = colN12
	}
	link := ""
	if links {
		link = r.MLflowURL
	}
	return s.cell(base, mlflowDisplay(r.MLflowURL), mlflowColWidth, fg, false, true, link)
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

// mlflowDisplay shortens an MLflow run URL to a "…/runs/<id-prefix>" label; the
// OSC 8 target keeps the full URL.
func mlflowDisplay(url string) string {
	id := mlflowRunID(url)
	if id == "" {
		return truncate(url, mlflowColWidth)
	}
	if len(id) > 8 {
		id = id[:8] + "…"
	}
	return "…/runs/" + id
}

// mlflowRunID extracts the run-id path segment from an MLflow URL.
func mlflowRunID(url string) string {
	_, after, ok := strings.Cut(url, "/runs/")
	if !ok {
		return ""
	}
	id, _, _ := strings.Cut(after, "/")
	return id
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
	if lipgloss.Width(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string([]rune(s)[:n-1]) + "…"
}
