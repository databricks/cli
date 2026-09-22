package aircmd

import "regexp"

// fatalPatterns match log lines that signal a run-ending failure (OOM, NCCL
// timeouts, CUDA errors, segfaults, etc.). In --json mode a matching line emits
// an ALERT event alongside its LOG event, giving an agent an immediate signal.
var fatalPatterns = []*regexp.Regexp{
	// OOM
	regexp.MustCompile(`(?i)CUDA out of memory`),
	regexp.MustCompile(`(?i)Out of memory: Kill(ed)? process`),
	// Signals
	regexp.MustCompile(`(?i)signal\s+(9|SIGKILL|SIGTERM)`),
	// NCCL / collective
	regexp.MustCompile(`Watchdog caught collective operation timeout`),
	regexp.MustCompile(`(?i)NCCL WARN .*(Conn|Net|IB|timeout|unhandled)`),
	regexp.MustCompile(`Got async error event`),
	regexp.MustCompile(`transport/net_ib\.cc:\d+.*WARN`),
	// CUDA
	regexp.MustCompile(`(?i)CUDA(?: runtime)? error`),
	regexp.MustCompile(`(?i)an illegal memory access was encountered`),
	regexp.MustCompile(`CUDA kernel errors might be asynchronously reported`),
	// Segfault
	regexp.MustCompile(`(?i)segmentation fault`),
	// Composer / llmfoundry specific
	regexp.MustCompile(`composer\.utils\..*Error`),
	regexp.MustCompile(`(?i)composer.*OutOfMemory`),
	// torch.distributed
	regexp.MustCompile(`torch\.distributed\..*(?:Error|Exception)`),
	regexp.MustCompile(`(?i)(?:TCPStore|Store).*timed?\s*out`),
	// GPU hardware (Xid from dmesg / driver)
	regexp.MustCompile(`Xid.*\b(48|63|64|79|94|95)\b`),
	// Streaming dataset
	regexp.MustCompile(`streaming\.base\..*(?:Error|Exception)`),
	// Bare "Killed" on its own line means OOM-killer or similar
	regexp.MustCompile(`^\s*Killed\s*$`),
	// MLflow stall — training has stopped logging metrics
	regexp.MustCompile(`\[MLflow Logger\]\[Warning\] No new logs have been emitted`),
	// The launch script prints this when the user's command fails; [1-9]\d* skips exit code 0.
	regexp.MustCompile(`ERROR: Script failed with exit code [1-9]\d* after \d+s`),
	// Missing command (exit 127), e.g. a typo.
	regexp.MustCompile(`(?i)command not found`),
}

// matchFatalPattern reports whether a log line matches a fatal-failure pattern.
func matchFatalPattern(line string) bool {
	for _, p := range fatalPatterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}
