package fs

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderBarWidth(t *testing.T) {
	// The bar must always occupy exactly barWidth cells regardless of ratio,
	// including the partial-cell and out-of-range cases.
	for _, ratio := range []float64{-0.5, 0, 0.0001, 0.123, 0.5, 0.52, 0.999, 1, 1.5} {
		if w := lipgloss.Width(renderBar(ratio)); w != barWidth {
			t.Errorf("renderBar(%v) width = %d, want %d", ratio, w, barWidth)
		}
	}
}

func TestRateMeter(t *testing.T) {
	var m rateMeter
	base := time.Unix(0, 0)

	if got := m.observe(base, 0); got != 0 {
		t.Fatalf("first observe = %v, want 0", got)
	}
	// A second sample at the same instant spans no interval to average over.
	if got := m.observe(base, 1<<20); got != 0 {
		t.Fatalf("zero-dt observe = %v, want 0 (no positive interval)", got)
	}

	// A steady 1 MiB/s stream averages to 1 MiB/s.
	var rate float64
	for i := 1; i <= 50; i++ {
		rate = m.observe(base.Add(time.Duration(i)*time.Second), int64(i)<<20)
	}
	const want = float64(1 << 20)
	if rate < want*0.99 || rate > want*1.01 {
		t.Errorf("steady rate = %v, want ~%v", rate, want)
	}
}

func TestRateMeterSmoothsBursts(t *testing.T) {
	var m rateMeter
	base := time.Unix(0, 0)

	// Prime a steady 1 MiB/s stream over the full window.
	for i := range 6 {
		m.observe(base.Add(time.Duration(i)*time.Second), int64(i)<<20)
	}

	// A large part lands almost instantly: 5 MiB in 100ms. A per-sample EMA
	// would read this as a ~50 MiB/s spike; the rolling window keeps the
	// reported rate near the windowed throughput.
	rate := m.observe(base.Add(5100*time.Millisecond), 10<<20)
	if rate > 3<<20 {
		t.Errorf("post-burst rate = %.0f B/s, want smoothed under %d B/s", rate, 3<<20)
	}
}

func TestFormatSpeed(t *testing.T) {
	cases := []struct {
		bps  float64
		want string
	}{
		{0, "0 B/s"},
		{-5, "0 B/s"},
		{512, "512 B/s"},
		{48 << 20, "48.0 MiB/s"},
	}
	for _, tc := range cases {
		if got := formatSpeed(tc.bps); got != tc.want {
			t.Errorf("formatSpeed(%v) = %q, want %q", tc.bps, got, tc.want)
		}
	}
}

func TestFormatETA(t *testing.T) {
	cases := []struct {
		remaining int64
		bps       float64
		want      string
	}{
		{0, 0, "--:--"},
		{1 << 20, 0, "--:--"},
		{-1, 1 << 20, "--:--"},
		{0, 1 << 20, "00:00"},
		{1 << 20, 1 << 20, "00:01"},
		{120 << 20, 1 << 20, "02:00"},
	}
	for _, tc := range cases {
		if got := formatETA(tc.remaining, tc.bps); got != tc.want {
			t.Errorf("formatETA(%d, %v) = %q, want %q", tc.remaining, tc.bps, got, tc.want)
		}
	}
}
