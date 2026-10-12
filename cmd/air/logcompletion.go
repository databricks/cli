package aircmd

import "strings"

// The remote entry script prints this block from its exit handler, as the last
// thing it does on each node:
//
//	================================
//	Execution Summary
//	================================
//	Total time: 32s
//	User code execution time: 32s
//	Completed at Fri Oct  9 09:46:18 PM UTC 2026
//	================================
//
// "Completed at" is printed only for exit code 0; a failure prints "ERROR: Script
// failed ..." instead, which is not treated as completion because the run may be
// retried. Jobs reports the run terminal tens of seconds after the block, so
// `air run --watch` uses it to stop as soon as every node has succeeded.
const (
	summaryHeader         = "Execution Summary"
	summaryRulePrefix     = "===="
	summaryTotalPrefix    = "Total time: "
	summaryUserCodePrefix = "User code execution time: "
	summarySuccessPrefix  = "Completed at "
)

// completionTailLines is how many of a node's newest log lines are searched for
// the summary block.
const completionTailLines = 20

// completionMatcher recognizes the entry script's success summary in one node's
// log lines, fed oldest-first. Only a consecutive block matches, so user output
// that happens to contain one of the lines does not.
type completionMatcher struct {
	step      int
	completed bool
}

func (m *completionMatcher) observe(line string) {
	line = strings.TrimSpace(line)
	switch {
	case line == summaryHeader:
		m.step = 1
	case m.step == 1 && strings.HasPrefix(line, summaryRulePrefix):
		m.step = 2
	case m.step == 2 && strings.HasPrefix(line, summaryTotalPrefix):
		m.step = 3
	case m.step == 3 && strings.HasPrefix(line, summaryUserCodePrefix):
		// Optional line between the total time and the outcome.
	case m.step == 3 && strings.HasPrefix(line, summarySuccessPrefix):
		m.completed = true
		m.step = 0
	default:
		m.step = 0
	}
}
