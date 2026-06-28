package fs

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/upload"
)

// renderThrottle caps how often the interactive bar is redrawn so a fast upload
// does not spend its time repainting the terminal; completion always renders.
const renderThrottle = 100 * time.Millisecond

// preparingFrame is how long each "Preparing upload" dot frame is shown before
// the upload reports its first completed part.
const preparingFrame = 400 * time.Millisecond

// barWidth keeps the rendered bar narrow enough that the whole progress line
// stays on a single terminal row, so the spinner's in-place redraw is clean.
const barWidth = 24

// rateWindow is the trailing duration over which the transfer rate is averaged.
// A few seconds smooths the burstiness of concurrent part completions while
// still tracking genuine changes in throughput.
const rateWindow = 5 * time.Second

// Bar styling. Green for the filled portion matches the cmdio spinner glyph;
// the remainder is dimmed.
var (
	barFilledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	barEmptyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// barBlocks are partial block glyphs for sub-character precision, so the bar
// advances smoothly rather than in whole-cell jumps. Index 0 is a full cell
// (8/8); index i is (8-i)/8 of a cell. Matches experimental/genie/agentstream.
var barBlocks = []string{"█", "▉", "▊", "▋", "▌", "▍", "▎", "▏"}

// humanBytes formats a byte count using binary (1024-based) units.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	div, exp := int64(unit), 0
	for n/div >= unit && exp < len(units)-1 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), units[exp])
}

// formatSpeed formats a transfer rate in bytes per second.
func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 0 {
		bytesPerSec = 0
	}
	return humanBytes(int64(bytesPerSec)) + "/s"
}

// formatETA formats the estimated time remaining as MM:SS. It returns a
// placeholder when the rate is unknown so the line does not show a misleading
// estimate before any throughput has been measured.
func formatETA(remaining int64, bytesPerSec float64) string {
	if bytesPerSec <= 0 || remaining < 0 {
		return "--:--"
	}
	secs := int(float64(remaining) / bytesPerSec)
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}

// formatPlainProgress renders a single non-interactive progress line.
func formatPlainProgress(p upload.Progress) string {
	if p.Total <= 0 {
		return "Uploaded " + humanBytes(p.Transferred)
	}
	pct := int(float64(p.Transferred) / float64(p.Total) * 100)
	return fmt.Sprintf("Uploaded %s / %s (%d%%)", humanBytes(p.Transferred), humanBytes(p.Total), pct)
}

// renderBar returns a fixed-width progress bar for the given completion ratio.
// The total visible width is always barWidth: full cells, an optional partial
// cell for the fractional remainder, then the dimmed empty track.
func renderBar(ratio float64) string {
	ratio = min(max(ratio, 0), 1)
	exact := ratio * barWidth
	full := int(exact)

	filled := strings.Repeat("█", full)
	partial := int((exact - float64(full)) * 8)
	if partial > 0 && full < barWidth {
		filled += barBlocks[8-partial]
		full++ // the partial glyph occupies one cell of the track
	}

	return barFilledStyle.Render(filled) +
		barEmptyStyle.Render(strings.Repeat("░", barWidth-full))
}

// rateSample is a cumulative byte count observed at a point in time.
type rateSample struct {
	at    time.Time
	bytes int64
}

// rateMeter computes the transfer rate as a rolling average over a trailing
// time window. Averaging over elapsed wall-clock time, rather than an
// exponential moving average over samples, keeps the reported speed and the
// ETA derived from it stable: the engine reports progress in irregular bursts
// as concurrent parts complete, and a per-sample EMA spikes on every burst.
// Callers pass the current time so the meter stays deterministic and
// unit-testable.
type rateMeter struct {
	samples []rateSample
}

// observe records a cumulative byte count at now and returns the average
// bytes/sec over the trailing rateWindow. It returns 0 until two samples span
// a positive interval (the first sample, or one taken after a stall longer
// than the window, leaves nothing to average against).
func (m *rateMeter) observe(now time.Time, transferred int64) float64 {
	// Drop samples that have aged out of the window, then record this one.
	// Filtering in place reuses the backing array; at the render cadence the
	// window holds at most a few dozen samples, so this stays cheap.
	cutoff := now.Add(-rateWindow)
	kept := m.samples[:0]
	for _, s := range m.samples {
		if !s.at.Before(cutoff) {
			kept = append(kept, s)
		}
	}
	m.samples = append(kept, rateSample{at: now, bytes: transferred})

	oldest := m.samples[0]
	dt := now.Sub(oldest.at).Seconds()
	if dt <= 0 {
		return 0
	}
	return float64(transferred-oldest.bytes) / dt
}

// progressRenderer builds the rich single-line progress string shown as the
// spinner suffix in an interactive terminal.
type progressRenderer struct {
	meter rateMeter
}

// newProgressRenderer creates a renderer.
func newProgressRenderer() *progressRenderer {
	return &progressRenderer{}
}

// render returns the progress line for the given progress sample at time now.
func (r *progressRenderer) render(now time.Time, p upload.Progress) string {
	rate := r.meter.observe(now, p.Transferred)
	if p.Total <= 0 {
		return fmt.Sprintf("%s  %s", humanBytes(p.Transferred), formatSpeed(rate))
	}
	ratio := float64(p.Transferred) / float64(p.Total)
	return fmt.Sprintf("%s %.0f%%  %s/%s  %s  ETA %s",
		renderBar(ratio), ratio*100,
		humanBytes(p.Transferred), humanBytes(p.Total),
		formatSpeed(rate), formatETA(p.Total-p.Transferred, rate))
}

// newProgressFunc returns the upload progress callback for the current terminal
// and a function that stops it. In an interactive terminal it renders a rich
// single-line bar via the spinner; otherwise it logs coarse progress so a long
// upload still shows life. The returned stop function is idempotent (it is safe
// to defer it and also call it before printing a summary line). The callback is
// serialized by the engine, so its captured state needs no locking.
func newProgressFunc(ctx context.Context) (upload.ProgressFunc, func()) {
	if cmdio.GetInteractiveMode(ctx) == cmdio.InteractiveModeNone {
		nextPct := 10
		fn := func(p upload.Progress) {
			// The final summary line covers completion; only log intermediate steps.
			if p.Total <= 0 || p.Transferred >= p.Total {
				return
			}
			pct := int(float64(p.Transferred) / float64(p.Total) * 100)
			if pct < nextPct {
				return
			}
			cmdio.LogString(ctx, formatPlainProgress(p))
			for pct >= nextPct {
				nextPct += 10
			}
		}
		return fn, func() {}
	}

	sp := cmdio.NewSpinner(ctx, cmdio.WithElapsedTime())
	r := newProgressRenderer()

	// The callback first fires only when a part completes; session initiation, URL
	// minting, and the first PUT take a beat, during which it never runs. Animate a
	// "Preparing upload" label with cycling dots over that window so the spinner is
	// not bare. mu serializes the animator's suffix updates with the callback's, and
	// preparing gates the animator off once real progress arrives, so the bar
	// replaces the label with no flicker.
	var (
		mu         sync.Mutex
		preparing  = true
		lastRender time.Time
	)
	prepCtx, stopPreparing := context.WithCancel(ctx)
	prepDone := make(chan struct{})
	go func() {
		defer close(prepDone)
		ticker := time.NewTicker(preparingFrame)
		defer ticker.Stop()
		for dots := 1; ; dots = dots%3 + 1 {
			mu.Lock()
			if !preparing {
				mu.Unlock()
				return
			}
			sp.Update("Preparing upload" + strings.Repeat(".", dots))
			mu.Unlock()
			select {
			case <-prepCtx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	fn := func(p upload.Progress) {
		mu.Lock()
		defer mu.Unlock()
		if preparing {
			preparing = false
			stopPreparing() // first real progress: stop the animator, the bar takes over
		}
		now := time.Now()
		if p.Total > 0 && p.Transferred < p.Total && now.Sub(lastRender) < renderThrottle {
			return
		}
		lastRender = now
		sp.Update(r.render(now, p))
	}
	// Stop the animator and wait for it to exit before closing the spinner, so no
	// suffix update races the spinner shutdown.
	return fn, func() {
		stopPreparing()
		<-prepDone
		sp.Close()
	}
}
