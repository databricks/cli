package agentstream

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// ANSI 256-color codes for chart series.
var seriesColors = []string{
	"\033[38;5;39m",  // blue
	"\033[38;5;208m", // orange
	"\033[38;5;82m",  // green
	"\033[38;5;196m", // red
	"\033[38;5;44m",  // cyan
	"\033[38;5;164m", // magenta
	"\033[38;5;226m", // yellow
	"\033[38;5;255m", // white
}

const (
	ansiDim   = "\033[2m"
	ansiBold  = "\033[1m"
	ansiReset = "\033[0m"
)

// Chart layout constants.
const (
	yLabelWidth   = 8  // characters reserved for y-axis labels
	minChartWidth = 20 // minimum usable chart width in characters
	legendMaxCols = 4  // max legend items per row
)

// dataSeries is a named series of float64 values.
type dataSeries struct {
	Name   string
	Values []float64
}

// RenderChart renders a terminal chart for the given visualization.
// Prints nothing if the data cannot be mapped to a renderable chart.
func RenderChart(w io.Writer, viz *VizEvent, width int) {
	if viz == nil || viz.Spec == nil || viz.Data == nil || len(viz.Data.Rows) == 0 {
		return
	}

	spec := viz.Spec
	data := viz.Data

	// Pre-check: only render if we can extract at least one series.
	series := extractSeries(spec, data)
	if len(series) == 0 {
		return
	}

	// Title.
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s%s%s\n", ansiBold, spec.Title, ansiReset)
	titleLen := min(len(spec.Title), width-4)
	if titleLen > 0 {
		fmt.Fprintf(w, "  %s%s%s\n", ansiDim, strings.Repeat("─", titleLen), ansiReset)
	}

	switch spec.WidgetType {
	case "bar":
		renderBarChart(w, spec, data, width)
	case "line", "area":
		renderLineChart(w, spec, data, width, spec.WidgetType == "area")
	default:
		renderBarChart(w, spec, data, width)
	}

	fmt.Fprintln(w)
}

// renderBarChart draws horizontal bars with ANSI colors.
func renderBarChart(w io.Writer, spec *VizSpec, data *TableData, width int) {
	xIdx := columnIndex(data.Columns, spec.XField)
	series := extractSeries(spec, data)
	if xIdx < 0 || len(series) == 0 {
		return
	}

	// Collect labels.
	labels := extractXLabels(data, xIdx, spec.ColorField != "")
	maxLabel := 0
	for _, l := range labels {
		maxLabel = max(maxLabel, len(l))
	}
	// Cap label width to prevent squishing the bars.
	if maxLabel > 30 {
		maxLabel = 30
	}

	// Find max value across all series.
	var maxVal float64
	for _, s := range series {
		for _, v := range s.Values {
			maxVal = math.Max(maxVal, v)
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	// Find max formatted value width for right-alignment.
	maxValLen := 0
	for _, s := range series {
		for _, v := range s.Values {
			maxValLen = max(maxValLen, len(formatNumber(v)))
		}
	}

	// Bar width (chars available for the bar itself).
	// Cap at 50 chars so bars stay readable on wide terminals.
	barWidth := width - maxLabel - maxValLen - 8
	if barWidth < minChartWidth {
		barWidth = minChartWidth
	}
	if barWidth > 50 {
		barWidth = 50
	}

	multiSeries := len(series) > 1
	nGroups := len(series[0].Values)

	// Use partial block characters for sub-character precision.
	// Full block = 8/8, then 7/8 through 1/8.
	blocks := []string{"█", "▉", "▊", "▋", "▌", "▍", "▎", "▏"}

	fmt.Fprintln(w)
	for gi := range nGroups {
		label := ""
		if gi < len(labels) {
			label = labels[gi]
			if len(label) > 30 {
				label = label[:27] + "..."
			}
		}

		for si, s := range series {
			if gi >= len(s.Values) {
				continue
			}
			v := s.Values[gi]

			// Label: only on first bar of group.
			if si == 0 {
				fmt.Fprintf(w, "  %*s ", maxLabel, label)
			} else {
				fmt.Fprintf(w, "  %*s ", maxLabel, "")
			}

			// Bar with sub-character precision.
			color := seriesColors[si%len(seriesColors)]
			exact := v / maxVal * float64(barWidth)
			if exact < 0 {
				exact = 0
			}
			full := int(exact)
			frac := exact - float64(full)
			partial := int(frac * 8)

			fmt.Fprint(w, color)
			fmt.Fprint(w, strings.Repeat("█", full))
			if partial > 0 && full < barWidth {
				fmt.Fprint(w, blocks[8-partial])
			}
			fmt.Fprint(w, ansiReset)

			// Value right-aligned.
			valStr := formatNumber(v)
			fmt.Fprintf(w, " %*s", maxValLen, valStr)
			if multiSeries {
				fmt.Fprintf(w, " %s(%s)%s", ansiDim, s.Name, ansiReset)
			}
			fmt.Fprintln(w)
		}
		if multiSeries && gi < nGroups-1 {
			fmt.Fprintln(w)
		}
	}

	if multiSeries {
		renderLegend(w, series)
	}
}

// renderLineChart draws line (or area) charts using braille characters.
func renderLineChart(w io.Writer, spec *VizSpec, data *TableData, width int, fill bool) {
	xIdx := columnIndex(data.Columns, spec.XField)
	series := extractSeries(spec, data)
	if xIdx < 0 || len(series) == 0 {
		return
	}

	// Chart dimensions. Cap width at 70 chars so charts stay
	// compact on wide terminals.
	cw := width - yLabelWidth - 4
	if cw < minChartWidth {
		cw = minChartWidth
	}
	if cw > 70 {
		cw = 70
	}
	// Height is roughly 40% of width for a pleasing aspect ratio.
	ch := max(10, cw*2/5)
	if ch > 25 {
		ch = 25
	}

	// Pixel dimensions (braille: 2 dots wide x 4 dots tall per char).
	pxW := cw * 2
	pxH := ch * 4

	// Y-axis range.
	var yMin, yMax float64
	for _, s := range series {
		for _, v := range s.Values {
			yMax = math.Max(yMax, v)
		}
	}
	if yMin == yMax {
		yMax = yMin + 1
	}
	// Add 5% headroom.
	yMax *= 1.05

	grid := newBrailleGrid(cw, ch)

	// Plot each series. Render in reverse order so the first series
	// (visually most important) wins color conflicts.
	for si := len(series) - 1; si >= 0; si-- {
		s := series[si]
		nPts := len(s.Values)
		if nPts == 0 {
			continue
		}

		for i := range nPts {
			x0 := mapRange(float64(i), 0, float64(max(nPts-1, 1)), 0, float64(pxW-1))
			y0 := mapRange(s.Values[i], yMin, yMax, float64(pxH-1), 0)

			if fill {
				px := int(math.Round(x0))
				py := int(math.Round(y0))
				for y := py; y < pxH; y++ {
					grid.set(px, y, si)
				}
			}

			if i < nPts-1 {
				x1 := mapRange(float64(i+1), 0, float64(max(nPts-1, 1)), 0, float64(pxW-1))
				y1 := mapRange(s.Values[i+1], yMin, yMax, float64(pxH-1), 0)
				ix0, iy0 := int(math.Round(x0)), int(math.Round(y0))
				ix1, iy1 := int(math.Round(x1)), int(math.Round(y1))
				// Draw a 2px thick line for better visibility.
				drawLine(grid, ix0, iy0, ix1, iy1, si)
				drawLine(grid, ix0, iy0+1, ix1, iy1+1, si)
			} else if nPts == 1 {
				grid.set(int(math.Round(x0)), int(math.Round(y0)), si)
			}
		}
	}

	// Render with y-axis labels.
	fmt.Fprintln(w)
	yLabels := computeYLabels(yMin, yMax, ch)
	for row := range ch {
		label := ""
		// Show labels on first row, last row, and evenly spaced.
		if row == 0 || row == ch-1 || row%(ch/4) == 0 {
			label = yLabels[row]
		}
		fmt.Fprintf(w, "  %*s ┤", yLabelWidth, label)
		for col := range cw {
			brChar := rune(0x2800) + grid.cells[row][col]
			ci := grid.colors[row][col]
			if ci >= 0 {
				fmt.Fprintf(w, "%s%c%s", seriesColors[ci%len(seriesColors)], brChar, ansiReset)
			} else {
				fmt.Fprintf(w, "%c", brChar)
			}
		}
		fmt.Fprintln(w)
	}

	// X-axis line.
	fmt.Fprintf(w, "  %*s └%s\n", yLabelWidth, "", strings.Repeat("─", cw))

	// X-axis labels.
	xLabels := extractXLabels(data, xIdx, spec.ColorField != "")
	renderXLabels(w, xLabels, cw, yLabelWidth+3)

	if len(series) > 1 {
		renderLegend(w, series)
	}
}

// --- Braille grid ---

type brailleGrid struct {
	width, height int      // in character cells
	cells         [][]rune // braille offset bits per cell
	colors        [][]int  // dominant series index per cell (-1 = none)
}

func newBrailleGrid(w, h int) *brailleGrid {
	cells := make([][]rune, h)
	colors := make([][]int, h)
	for i := range h {
		cells[i] = make([]rune, w)
		colors[i] = make([]int, w)
		for j := range w {
			colors[i][j] = -1
		}
	}
	return &brailleGrid{width: w, height: h, cells: cells, colors: colors}
}

// Braille dot layout within a 2x4 cell:
//
//	col 0    col 1
//	bit 0    bit 3    row 0
//	bit 1    bit 4    row 1
//	bit 2    bit 5    row 2
//	bit 6    bit 7    row 3
var brailleBits = [2][4]rune{
	{0x01, 0x02, 0x04, 0x40}, // left column
	{0x08, 0x10, 0x20, 0x80}, // right column
}

func (g *brailleGrid) set(px, py, series int) {
	cx, cy := px/2, py/4
	if cx < 0 || cx >= g.width || cy < 0 || cy >= g.height {
		return
	}
	dx, dy := px%2, py%4
	g.cells[cy][cx] |= brailleBits[dx][dy]
	g.colors[cy][cx] = series
}

// drawLine plots a line between two pixel coordinates using Bresenham's algorithm.
func drawLine(g *brailleGrid, x0, y0, x1, y1, series int) {
	dx := intAbs(x1 - x0)
	dy := intAbs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	for {
		g.set(x0, y0, series)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// --- Data extraction ---

// extractSeries converts table data into typed series based on the viz spec.
func extractSeries(spec *VizSpec, data *TableData) []dataSeries {
	xIdx := columnIndex(data.Columns, spec.XField)
	if xIdx < 0 {
		return nil
	}

	if spec.ColorField != "" && len(spec.YFields) > 0 {
		colorIdx := columnIndex(data.Columns, spec.ColorField)
		yIdx := columnIndex(data.Columns, spec.YFields[0])
		if colorIdx >= 0 && yIdx >= 0 {
			return extractLongFormat(data, xIdx, yIdx, colorIdx)
		}
	}

	var result []dataSeries
	for _, yf := range spec.YFields {
		yi := columnIndex(data.Columns, yf)
		if yi < 0 {
			continue
		}
		s := dataSeries{Name: yf}
		for _, row := range data.Rows {
			s.Values = append(s.Values, parseFloat(row[yi]))
		}
		result = append(result, s)
	}
	return result
}

// extractLongFormat pivots long-format data (one row per x/category combo) into series.
func extractLongFormat(data *TableData, xIdx, yIdx, colorIdx int) []dataSeries {
	var xValues []string
	xSeen := map[string]bool{}
	for _, row := range data.Rows {
		x := row[xIdx]
		if !xSeen[x] {
			xValues = append(xValues, x)
			xSeen[x] = true
		}
	}

	type seriesData struct {
		name   string
		values map[string]float64
	}
	var seriesOrder []string
	seriesMap := map[string]*seriesData{}

	for _, row := range data.Rows {
		sName := row[colorIdx]
		if seriesMap[sName] == nil {
			seriesMap[sName] = &seriesData{name: sName, values: map[string]float64{}}
			seriesOrder = append(seriesOrder, sName)
		}
		seriesMap[sName].values[row[xIdx]] = parseFloat(row[yIdx])
	}

	var result []dataSeries
	for _, name := range seriesOrder {
		sd := seriesMap[name]
		s := dataSeries{Name: sd.name}
		for _, x := range xValues {
			s.Values = append(s.Values, sd.values[x])
		}
		result = append(result, s)
	}
	return result
}

// --- Helpers ---

// ParseMarkdownTable extracts column names and rows from a markdown table.
func ParseMarkdownTable(text string) *TableData {
	lines := strings.Split(text, "\n")
	var tableLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "|") {
			tableLines = append(tableLines, line)
		}
	}
	if len(tableLines) < 3 {
		return nil
	}

	columns := splitTableRow(tableLines[0])
	var rows [][]string
	for _, line := range tableLines[2:] {
		row := splitTableRow(line)
		if len(row) == len(columns) {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return &TableData{Columns: columns, Rows: rows}
}

func splitTableRow(line string) []string {
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && !strings.HasPrefix(p, "---") {
			cells = append(cells, p)
		}
	}
	return cells
}

func columnIndex(columns []string, name string) int {
	for i, c := range columns {
		if c == name {
			return i
		}
	}
	return -1
}

func uniqueXLabels(data *TableData, xIdx int) []string {
	var labels []string
	seen := map[string]bool{}
	for _, row := range data.Rows {
		l := row[xIdx]
		if !seen[l] {
			labels = append(labels, l)
			seen[l] = true
		}
	}
	return labels
}

func extractXLabels(data *TableData, xIdx int, longFormat bool) []string {
	if longFormat {
		return uniqueXLabels(data, xIdx)
	}
	labels := make([]string, len(data.Rows))
	for i, row := range data.Rows {
		labels[i] = row[xIdx]
	}
	return labels
}

func parseFloat(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func formatNumber(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		s := strconv.FormatInt(int64(v), 10)
		return addThousandSep(s)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

func addThousandSep(s string) string {
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}
	n := len(s)
	if n <= 3 {
		if negative {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	first := n % 3
	if first > 0 {
		b.WriteString(s[:first])
	}
	for i := first; i < n; i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	if negative {
		return "-" + b.String()
	}
	return b.String()
}

func computeYLabels(_ float64, yMax float64, rows int) []string {
	labels := make([]string, rows)
	for i := range rows {
		v := yMax - yMax*float64(i)/float64(rows-1)
		labels[i] = formatNumber(v)
	}
	return labels
}

func renderXLabels(w io.Writer, labels []string, chartWidth, leftPad int) {
	if len(labels) == 0 {
		return
	}

	maxLabels := chartWidth / 12
	if maxLabels < 2 {
		maxLabels = 2
	}
	step := max(1, (len(labels)-1)/(maxLabels-1))

	fmt.Fprintf(w, "%*s", leftPad, "")
	pos := 0
	for i := 0; i < len(labels); i += step {
		label := labels[i]
		if len(label) > 10 {
			label = label[:10]
		}
		target := int(float64(i) / float64(max(len(labels)-1, 1)) * float64(chartWidth-1))
		for pos < target {
			fmt.Fprint(w, " ")
			pos++
		}
		fmt.Fprint(w, label)
		pos += len(label)
	}
	fmt.Fprintln(w)
}

func renderLegend(w io.Writer, series []dataSeries) {
	fmt.Fprintln(w)
	col := 0
	for si, s := range series {
		color := seriesColors[si%len(seriesColors)]
		entry := fmt.Sprintf("  %s●%s %s", color, ansiReset, s.Name)
		fmt.Fprint(w, entry)
		col++
		if col >= legendMaxCols && si < len(series)-1 {
			fmt.Fprintln(w)
			col = 0
		}
	}
	fmt.Fprintln(w)
}

func mapRange(v, inMin, inMax, outMin, outMax float64) float64 {
	if inMax == inMin {
		return (outMin + outMax) / 2
	}
	return outMin + (v-inMin)/(inMax-inMin)*(outMax-outMin)
}

func intAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
