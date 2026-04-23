package textutil

// LevenshteinDistance computes the edit distance between two strings using
// a space-optimized dynamic programming approach.
func LevenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Use a single row for the DP table.
	prev := make([]int, len(b)+1)
	for j := range len(b) + 1 {
		prev[j] = j
	}

	for i := range len(a) {
		curr := make([]int, len(b)+1)
		curr[0] = i + 1
		for j := range len(b) {
			cost := 1
			if a[i] == b[j] {
				cost = 0
			}
			curr[j+1] = min(
				curr[j]+1,    // insertion
				prev[j+1]+1,  // deletion
				prev[j]+cost, // substitution
			)
		}
		prev = curr
	}

	return prev[len(b)]
}
