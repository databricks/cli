package agentstream

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
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

// Widget types from the visualization spec.
const (
	widgetBar  = "bar"
	widgetLine = "line"
	widgetArea = "area"
)

// Chart layout constants.
const (
	yLabelWidth   = 8  // characters reserved for y-axis labels
	minChartWidth = 20 // minimum usable chart width in characters
	legendMaxCols = 4  // max legend items per row
	maxLabelRunes = 30 // x labels longer than this are truncated
)

// chartStyle holds the ANSI sequences used by a render pass. When color is
// disabled every sequence is empty, so piped or redirected output stays free
// of escape codes.
type chartStyle struct {
	dim, bold, reset string
	series           []string
}

func newChartStyle(color bool) chartStyle {
	if !color {
		return chartStyle{series: make([]string, len(seriesColors))}
	}
	return chartStyle{dim: ansiDim, bold: ansiBold, reset: ansiReset, series: seriesColors}
}

// dataSeries is a named series of float64 values.
type dataSeries struct {
	Name   string
	Values []float64
}

// RenderChart renders a terminal chart for the given visualization and
// reports whether anything was drawn, so the caller can show a placeholder
// instead of silently dropping a chart the answer text refers to.
func RenderChart(w io.Writer, viz *VizEvent, width int, color bool) bool {
	if viz == nil || viz.Spec == nil || viz.Data == nil || len(viz.Data.Rows) == 0 {
		return false
	}

	spec := viz.Spec

	// Drop rows whose y values cannot be parsed instead of plotting them as
	// zero: a fabricated zero bar misrepresents the data, which is worse than
	// a missing row.
	data := filterPlottableRows(spec, viz.Data)
	if data == nil || len(data.Rows) == 0 {
		return false
	}

	series := extractSeries(spec, data)
	if len(series) == 0 {
		return false
	}

	st := newChartStyle(color)

	// Title.
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s%s%s\n", st.bold, spec.Title, st.reset)
	titleLen := min(len(spec.Title), width-4)
	if titleLen > 0 {
		fmt.Fprintf(w, "  %s%s%s\n", st.dim, strings.Repeat("─", titleLen), st.reset)
	}

	switch spec.WidgetType {
	case widgetLine, widgetArea:
		renderLineChart(w, spec, data, series, width, spec.WidgetType == widgetArea, st)
	default:
		// Bar is also the fallback for unknown widget types: bars stay
		// readable for any categorical data.
		renderBarChart(w, spec, data, series, width, st)
	}

	fmt.Fprintln(w)
	return true
}

// filterPlottableRows returns a copy of data containing only rows whose y
// values all parse as finite numbers. Returns nil when no y field matches the
// columns at all, so spec/column mismatches degrade to "no chart" rather than
// a chart of zeros.
func filterPlottableRows(spec *VizSpec, data *TableData) *TableData {
	var yIdx []int
	for _, yf := range spec.YFields {
		if i := columnIndex(data.Columns, yf); i >= 0 {
			yIdx = append(yIdx, i)
		}
	}
	if len(yIdx) == 0 {
		return nil
	}

	rows := make([][]string, 0, len(data.Rows))
	for _, row := range data.Rows {
		ok := true
		for _, i := range yIdx {
			if _, parsed := parseFloat(rowString(row, i)); !parsed {
				ok = false
				break
			}
		}
		if ok {
			rows = append(rows, row)
		}
	}
	return &TableData{Columns: data.Columns, Rows: rows}
}

// renderBarChart draws horizontal bars. The caller guarantees series is
// non-empty, which extractSeries only produces when XField exists, so xIdx
// is never negative here.
func renderBarChart(w io.Writer, spec *VizSpec, data *TableData, series []dataSeries, width int, st chartStyle) {
	xIdx := columnIndex(data.Columns, spec.XField)

	// Collect labels. Measure and truncate in runes: byte-based slicing can
	// split a multi-byte character, and fmt pads %*s by rune count.
	labels := extractXLabels(data, xIdx, spec.ColorField != "")
	maxLabel := 0
	for _, l := range labels {
		maxLabel = max(maxLabel, utf8.RuneCountInString(l))
	}
	// Cap label width to prevent squishing the bars.
	maxLabel = min(maxLabel, maxLabelRunes)

	// Bars scale by absolute value so negative aggregates remain visible;
	// the signed number beside each bar carries the direction.
	var maxVal float64
	for _, s := range series {
		for _, v := range s.Values {
			maxVal = math.Max(maxVal, math.Abs(v))
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
	barWidth := min(max(width-maxLabel-maxValLen-8, minChartWidth), 50)

	multiSeries := len(series) > 1
	nGroups := len(series[0].Values)

	// Use partial block characters for sub-character precision.
	// Full block = 8/8, then 7/8 through 1/8.
	blocks := []string{"█", "▉", "▊", "▋", "▌", "▍", "▎", "▏"}

	fmt.Fprintln(w)
	for gi := range nGroups {
		label := ""
		if gi < len(labels) {
			label = truncateRunes(labels[gi], maxLabelRunes)
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
			color := st.series[si%len(st.series)]
			exact := math.Abs(v) / maxVal * float64(barWidth)
			full := int(exact)
			frac := exact - float64(full)
			partial := int(frac * 8)

			fmt.Fprint(w, color)
			fmt.Fprint(w, strings.Repeat("█", full))
			if partial > 0 && full < barWidth {
				fmt.Fprint(w, blocks[8-partial])
			}
			fmt.Fprint(w, st.reset)

			// Value right-aligned.
			valStr := formatNumber(v)
			fmt.Fprintf(w, " %*s", maxValLen, valStr)
			if multiSeries {
				fmt.Fprintf(w, " %s(%s)%s", st.dim, s.Name, st.reset)
			}
			fmt.Fprintln(w)
		}
		if multiSeries && gi < nGroups-1 {
			fmt.Fprintln(w)
		}
	}

	if multiSeries {
		renderLegend(w, series, st)
	}
}

// renderLineChart draws line (or area) charts using braille characters. The
// caller guarantees series is non-empty, which extractSeries only produces
// when XField exists, so xIdx is never negative here.
func renderLineChart(w io.Writer, spec *VizSpec, data *TableData, series []dataSeries, width int, fill bool, st chartStyle) {
	xIdx := columnIndex(data.Columns, spec.XField)

	// Chart dimensions. Cap width at 70 chars so charts stay
	// compact on wide terminals.
	cw := min(max(width-yLabelWidth-4, minChartWidth), 70)
	// Height is roughly 40% of width for a pleasing aspect ratio.
	ch := min(max(10, cw*2/5), 25)

	// Pixel dimensions (braille: 2 dots wide x 4 dots tall per char).
	pxW := cw * 2
	pxH := ch * 4

	// Y-axis range.
	yMin := math.Inf(1)
	yMax := math.Inf(-1)
	for _, s := range series {
		for _, v := range s.Values {
			yMin = math.Min(yMin, v)
			yMax = math.Max(yMax, v)
		}
	}
	if math.IsInf(yMin, 1) {
		return
	}
	if yMin > 0 {
		yMin = 0
	}
	if yMax < 0 {
		yMax = 0
	}
	if yMin == yMax {
		yMax = yMin + 1
	}
	span := yMax - yMin
	yMax += span * 0.05
	if yMin < 0 {
		yMin -= span * 0.05
	}

	grid := newBrailleGrid(cw, ch)

	// Plot each series. Render in reverse order so the first series
	// (visually most important) wins color conflicts.
	for si, s := range slices.Backward(series) {
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
				fmt.Fprintf(w, "%s%c%s", st.series[ci%len(st.series)], brChar, st.reset)
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
		renderLegend(w, series, st)
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
			// Rows are pre-filtered by filterPlottableRows, so the parse
			// cannot fail here.
			v, _ := parseFloat(rowString(row, yi))
			s.Values = append(s.Values, v)
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
		x, ok := rowValue(row, xIdx)
		if !ok {
			continue
		}
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
		x, ok := rowValue(row, xIdx)
		if !ok {
			continue
		}
		sName, ok := rowValue(row, colorIdx)
		if !ok {
			continue
		}
		if seriesMap[sName] == nil {
			seriesMap[sName] = &seriesData{name: sName, values: map[string]float64{}}
			seriesOrder = append(seriesOrder, sName)
		}
		// Rows are pre-filtered by filterPlottableRows, so the parse cannot
		// fail here.
		v, _ := parseFloat(rowString(row, yIdx))
		seriesMap[sName].values[x] = v
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

func columnIndex(columns []string, name string) int {
	for i, c := range columns {
		if c == name {
			return i
		}
	}
	return -1
}

func rowValue(row []string, idx int) (string, bool) {
	if idx < 0 || idx >= len(row) {
		return "", false
	}
	return row[idx], true
}

func rowString(row []string, idx int) string {
	v, _ := rowValue(row, idx)
	return v
}

func uniqueXLabels(data *TableData, xIdx int) []string {
	var labels []string
	seen := map[string]bool{}
	for _, row := range data.Rows {
		l, ok := rowValue(row, xIdx)
		if !ok {
			continue
		}
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
		labels[i] = rowString(row, xIdx)
	}
	return labels
}

// parseFloat parses a table cell as a finite float. The second return value
// distinguishes "zero" from "not a number at all" so unparseable cells can be
// dropped instead of plotted as zero.
func parseFloat(s string) (float64, bool) {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// truncateRunes shortens s to at most n runes, ending with "..." when
// truncated. Slicing runes rather than bytes keeps multi-byte characters
// intact.
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-3]) + "..."
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

func computeYLabels(yMin, yMax float64, rows int) []string {
	labels := make([]string, rows)
	if rows == 1 {
		labels[0] = formatNumber(yMax)
		return labels
	}
	for i := range rows {
		v := yMax - (yMax-yMin)*float64(i)/float64(rows-1)
		labels[i] = formatNumber(v)
	}
	return labels
}

func renderXLabels(w io.Writer, labels []string, chartWidth, leftPad int) {
	if len(labels) == 0 {
		return
	}

	maxLabels := max(chartWidth/12, 2)
	step := max(1, (len(labels)-1)/(maxLabels-1))

	fmt.Fprintf(w, "%*s", leftPad, "")
	pos := 0
	for i := 0; i < len(labels); i += step {
		label := truncateRunes(labels[i], 10)
		target := int(float64(i) / float64(max(len(labels)-1, 1)) * float64(chartWidth-1))
		for pos < target {
			fmt.Fprint(w, " ")
			pos++
		}
		fmt.Fprint(w, label)
		pos += utf8.RuneCountInString(label)
	}
	fmt.Fprintln(w)
}

func renderLegend(w io.Writer, series []dataSeries, st chartStyle) {
	fmt.Fprintln(w)
	col := 0
	for si, s := range series {
		color := st.series[si%len(st.series)]
		entry := fmt.Sprintf("  %s●%s %s", color, st.reset, s.Name)
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
