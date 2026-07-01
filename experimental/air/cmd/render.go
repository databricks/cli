package aircmd

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/muesli/termenv"
	"go.yaml.in/yaml/v3"
)

// Box titles, rendered into the top border of each box.
const (
	configBoxTitle   = "Configuration"
	metadataBoxTitle = "Metadata"
)

// minBoxInnerWidth keeps all boxes a uniform, comfortable width; boxHPad and
// boxVPad are the horizontal and vertical padding inside each box.
const (
	minBoxInnerWidth = 60
	boxHPad          = 2
	boxVPad          = 1
)

// palette holds the lipgloss styles for the single-run view. Two layers: a
// neutral ramp for chrome and text, and an accent palette for syntax and data.
// All styles come from one renderer, so they honor its color profile (Ascii
// under --no-color / non-TTY, which strips every escape).
type palette struct {
	n7  lipgloss.Style // dim: block indicator, empty progress, command-block "|"
	n8  lipgloss.Style // muted: field labels
	n12 lipgloss.Style // content: config and metadata values, percent

	border lipgloss.Style // box borders and titles

	blue  lipgloss.Style // yaml keys and hyperlinks
	green lipgloss.Style // success status and progress fill
	amber lipgloss.Style // in-progress status
	red   lipgloss.Style // failed status
}

func newPalette(r *lipgloss.Renderer) palette {
	fg := func(hex string) lipgloss.Style { return r.NewStyle().Foreground(lipgloss.Color(hex)) }
	return palette{
		n7:     fg("#6E6E70"),
		n8:     fg("#8C8A86"),
		n12:    fg("#F9F7F4"), // Oat Light
		border: fg("#B7A8E8"), // light purple (box borders and titles)
		blue:   fg("#8FB3DC"),
		green:  fg("#74C39A"),
		amber:  fg("#DCAA5C"),
		red:    fg("#D9756B"),
	}
}

// runView is the resolved, display-ready data the renderer draws. It is built
// from getData plus the MLflow enrichment, so the renderer itself does no API
// calls or formatting decisions.
type runView struct {
	runID        string
	dashboardURL string
	status       string
	submitted    string
	retries      int
	maxRetries   string
	duration     string
	experiment   string
	mlflowLabel  string
	mlflowURL    string
	user         string
	accelerators string
	environment  string
}

// renderRunText writes the styled single-run view: a training-config box, a
// completed progress bar, and a field list, separated by blank lines. It is a
// one-shot renderer — it builds the full string and writes it once, with no
// streaming, spinner, or redraw.
func renderRunText(ctx context.Context, out io.Writer, w *databricks.WorkspaceClient, run *jobs.Run, data *getData, ids *mlflowIdentifiers) {
	colorOn := cmdio.SupportsColor(ctx, out)
	renderer := lipgloss.NewRenderer(out)
	if !colorOn {
		// Ascii emits no SGR codes; combined with the link fallback below this
		// gives clean, un-escaped output under --no-color / NO_COLOR / piped stdout.
		renderer.SetColorProfile(termenv.Ascii)
	}
	p := newPalette(renderer)

	view := runView{
		runID:        data.RunID,
		dashboardURL: data.DashboardURL,
		status:       data.Status,
		submitted:    data.SubmittedDisplay,
		retries:      data.AttemptNumber,
		maxRetries:   data.MaxRetriesDisplay,
		duration:     data.DurationDisplay,
		experiment:   data.ExperimentDisplay,
		mlflowLabel:  na,
		user:         data.UserDisplay,
		accelerators: data.AcceleratorsDisplay,
		environment:  data.EnvironmentDisplay,
	}

	if ids != nil {
		view.mlflowLabel = mlflowRunLabel(fetchMLflowRunName(ctx, w, ids.RunID), ids.RunID)
		view.mlflowURL = mlflowRunURL(w.Config.Host, ids)
	}

	var sections []string
	if task := genAIComputeTask(run); task != nil {
		if body := colorizeConfig(p, configYAML(ctx, w, task)); body != "" {
			sections = append(sections, renderBox(p, configBoxTitle, body))
		}
	}
	sections = append(sections, renderBox(p, metadataBoxTitle, renderFields(p, colorOn, view)))

	// A single write: a blank line before the first box and after the last, and
	// one between each box.
	fmt.Fprintf(out, "\n%s\n\n", strings.Join(sections, "\n\n"))

	// Bare-URL footer so the job run / MLflow links remain reachable when
	// stdout is not a hyperlink-capable terminal (piped, redirected, NO_COLOR).
	// In that case the OSC 8 hyperlinks on the Run ID / MLflow Run cells
	// degrade to plain labels and the URLs would otherwise disappear from text
	// output, breaking workflows like `air get X > out.txt` or
	// `NO_COLOR=1 air get X` that the previous `Job Link:` line supported.
	if view.dashboardURL != "" {
		fmt.Fprintf(out, "Run URL:    %s\n", view.dashboardURL)
	}
	if view.mlflowURL != "" {
		fmt.Fprintf(out, "MLflow URL: %s\n", view.mlflowURL)
	}
}

// genAIComputeTask returns the run's first GenAI-compute task, or nil.
func genAIComputeTask(run *jobs.Run) *jobs.GenAiComputeTask {
	if len(run.Tasks) == 0 {
		return nil
	}
	return run.Tasks[0].GenAiComputeTask
}

// configYAML returns the run's resolved training config as YAML for the box. The
// full config (including the command/script) lives in the run's parameters, not
// the structured task fields, so we prefer the inline parameters, then the
// parameters file, and only synthesize a minimal config as a last resort.
func configYAML(ctx context.Context, w *databricks.WorkspaceClient, task *jobs.GenAiComputeTask) string {
	if task.YamlParameters != "" {
		return strings.TrimRight(reformatYAMLForDisplay([]byte(task.YamlParameters)), "\n")
	}
	if task.YamlParametersFilePath != "" {
		if raw := downloadConfig(ctx, w, task.YamlParametersFilePath); len(raw) > 0 {
			return strings.TrimRight(reformatYAMLForDisplay(raw), "\n")
		}
	}
	return synthConfigYAML(task)
}

// downloadConfig fetches the run's training-config file, returning nil on
// failure (logged as a warning). Best-effort, like the rest of the enrichment.
func downloadConfig(ctx context.Context, w *databricks.WorkspaceClient, path string) []byte {
	r, err := w.Workspace.Download(ctx, path)
	if err != nil {
		log.Warnf(ctx, "air get: could not download training config %s: %v", path, err)
		return nil
	}
	defer r.Close()
	content, err := io.ReadAll(r)
	if err != nil {
		log.Warnf(ctx, "air get: could not read training config %s: %v", path, err)
		return nil
	}
	return content
}

// configBox describes the synthesized config we marshal when the run exposes no
// parameters, in the order the fields are shown.
type configBox struct {
	ExperimentName string         `yaml:"experiment_name,omitempty"`
	Compute        *configCompute `yaml:"compute,omitempty"`
}

type configCompute struct {
	AcceleratorType string `yaml:"accelerator_type,omitempty"`
	NumAccelerators int    `yaml:"num_accelerators,omitempty"`
}

// synthConfigYAML builds a minimal config from the structured task fields. It
// omits the command, which is only available in the run parameters.
func synthConfigYAML(task *jobs.GenAiComputeTask) string {
	cfg := configBox{}
	if task.MlflowExperimentName != "" {
		cfg.ExperimentName = stripExperimentUserPrefix(task.MlflowExperimentName)
	}
	if task.Compute != nil && task.Compute.NumGpus > 0 {
		cfg.Compute = &configCompute{
			AcceleratorType: task.Compute.GpuType,
			NumAccelerators: task.Compute.NumGpus,
		}
	}
	if cfg.ExperimentName == "" && cfg.Compute == nil {
		return ""
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return ""
	}
	return strings.TrimRight(reformatYAMLForDisplay(b), "\n")
}

// colorizeConfig styles a YAML config block line by line.
func colorizeConfig(p palette, body string) string {
	if body == "" {
		return ""
	}
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		lines[i] = colorizeConfigLine(p, line)
	}
	return strings.Join(lines, "\n")
}

// colorizeConfigLine colors one YAML line: keys blue, the `|` block indicator
// dim, and every value (and the command body that isn't a `key:` pair) in the
// neutral content color.
func colorizeConfigLine(p palette, line string) string {
	indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
	trimmed := strings.TrimLeft(line, " ")

	if i := strings.IndexByte(trimmed, ':'); i > 0 && isConfigKey(trimmed[:i]) {
		key := trimmed[:i]
		value := strings.TrimSpace(trimmed[i+1:])
		styled := indent + p.blue.Render(key+":")
		switch value {
		case "":
			// A mapping parent such as "compute:" has no value of its own.
		case "|":
			styled += " " + p.n7.Render(value)
		default:
			styled += " " + p.n12.Render(value)
		}
		return styled
	}
	return indent + p.n12.Render(trimmed)
}

// isConfigKey reports whether s is a bare YAML key (lowercase, digits, and
// underscores). It guards against treating a colon inside a command body as a
// key/value separator.
func isConfigKey(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r != '_' && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// renderBox draws a rounded-border box around body, with title rendered into the
// top border in the border color. body lines are padded to the widest one (or
// minBoxInnerWidth), with boxHPad columns and boxVPad rows of padding inside.
func renderBox(p palette, title, body string) string {
	border := lipgloss.RoundedBorder()
	lines := strings.Split(body, "\n")
	pad := strings.Repeat(" ", boxHPad)

	titleWidth := lipgloss.Width(title)
	inner := max(minBoxInnerWidth, titleWidth+2)
	for _, line := range lines {
		inner = max(inner, lipgloss.Width(line))
	}

	left := p.border.Render(border.Left)
	right := p.border.Render(border.Right)
	blank := left + strings.Repeat(" ", inner+2*boxHPad) + right

	var b strings.Builder
	// Top: ╭─ <title> ──…──╮. The dash count makes the row width match the body,
	// accounting for the boxHPad columns on each side.
	trailing := inner + 2*boxHPad - titleWidth - 3
	b.WriteString(p.border.Render(border.TopLeft + border.Top))
	b.WriteString(" " + p.border.Render(title) + " ")
	b.WriteString(p.border.Render(strings.Repeat(border.Top, trailing) + border.TopRight))
	b.WriteByte('\n')

	for range boxVPad {
		b.WriteString(blank + "\n")
	}
	for _, line := range lines {
		fill := strings.Repeat(" ", inner-lipgloss.Width(line))
		b.WriteString(left + pad + line + fill + pad + right)
		b.WriteByte('\n')
	}
	for range boxVPad {
		b.WriteString(blank + "\n")
	}

	b.WriteString(p.border.Render(border.BottomLeft + strings.Repeat(border.Bottom, inner+2*boxHPad) + border.BottomRight))
	return b.String()
}

// renderFields draws the two-column summary: muted labels right-padded to the
// longest one, neutral values, a status-colored Status, and blue Run ID / MLflow
// Run hyperlinks.
func renderFields(p palette, colorOn bool, v runView) string {
	status := statusStyle(p, v.status).Render("● " + v.status)
	rows := []string{
		field(p, "Run ID", link(colorOn, p.blue, v.runID, v.dashboardURL)),
		field(p, "Status", status),
		field(p, "Submitted", p.n12.Render(v.submitted)),
		field(p, "Retries", p.n12.Render(strconv.Itoa(v.retries))),
		field(p, "Max Retries", p.n12.Render(v.maxRetries)),
		field(p, "Duration", p.n12.Render(v.duration)),
		field(p, "Experiment", p.n12.Render(v.experiment)),
		field(p, "MLflow Run", link(colorOn, p.blue, v.mlflowLabel, v.mlflowURL)),
		field(p, "User", p.n12.Render(v.user)),
		field(p, "Accelerators", p.n12.Render(v.accelerators)),
		field(p, "Environment", p.n12.Render(v.environment)),
	}
	return strings.Join(rows, "\n")
}

// fieldLabelWidth is the width of the longest label ("Accelerators"), so values
// line up in a single column.
const fieldLabelWidth = len("Accelerators")

func field(p palette, label, value string) string {
	return p.n8.Render(label+strings.Repeat(" ", fieldLabelWidth-len(label))) + "  " + value
}

// link renders label as an OSC 8 terminal hyperlink to url in the given style
// (underlined). With color off (or no url) it is just the styled label so the
// box stays aligned; the URLs remain available in JSON output.
func link(colorOn bool, style lipgloss.Style, label, url string) string {
	if !colorOn || url == "" {
		return style.Render(label)
	}
	// Wrap the already-styled label in the hyperlink. Passing the OSC 8 escape
	// through lipgloss.Render instead corrupts it: lipgloss re-styles each rune
	// and splits the "\x1b]8;;" introducer, so the terminal can't parse the
	// sequence and prints it literally.
	return termenv.Hyperlink(url, style.Underline(true).Render(label))
}

// statusStyle maps a run status to its accent color: green for success, red for
// terminal failures, amber for everything still in flight.
func statusStyle(p palette, status string) lipgloss.Style {
	switch {
	case isSuccessStatus(status):
		return p.green
	case isFailedStatus(status):
		return p.red
	default:
		return p.amber
	}
}

func isSuccessStatus(status string) bool {
	return status == "SUCCESS"
}

func isFailedStatus(status string) bool {
	switch status {
	case "FAILED", "TIMEDOUT", "CANCELED", "INTERNAL_ERROR", "UPSTREAM_FAILED", "UPSTREAM_CANCELED":
		return true
	}
	return false
}
