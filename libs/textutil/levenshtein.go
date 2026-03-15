package textutil

// Levenshtein computes the edit distance between two strings using single-row DP.
func Levenshtein(a, b string) int {
	if len(a) < len(b) {
		a, b = b, a
	}

	row := make([]int, len(b)+1)
	for j := range row {
		row[j] = j
	}

	for i := range len(a) {
		prev := row[0]
		row[0] = i + 1
		for j := range len(b) {
			cost := 1
			if a[i] == b[j] {
				cost = 0
			}
			tmp := row[j+1]
			row[j+1] = min(
				row[j+1]+1, // deletion
				row[j]+1,   // insertion
				prev+cost,  // substitution
			)
			prev = tmp
		}
	}

	return row[len(b)]
}

// ClosestMatch finds the candidate with the smallest edit distance to key,
// within a threshold of min(3, max(1, len(key)/2)). Returns ("", 0) if no
// candidate is within threshold.
func ClosestMatch(key string, candidates []string) (string, int) {
	threshold := min(3, max(1, len(key)/2))
	bestDist := threshold + 1
	bestMatch := ""

	for _, c := range candidates {
		d := Levenshtein(key, c)
		if d < bestDist {
			bestDist = d
			bestMatch = c
		}
	}

	if bestMatch == "" {
		return "", 0
	}
	return bestMatch, bestDist
}
