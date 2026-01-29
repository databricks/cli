package sdkdocs

import (
	"sort"
	"strings"
)

// SearchResult represents a single search result.
type SearchResult struct {
	Type        string  `json:"type"` // "service", "method", "type", "enum"
	Name        string  `json:"name"`
	Path        string  `json:"path"`
	Service     string  `json:"service,omitempty"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

// SearchOptions configures the search behavior.
type SearchOptions struct {
	Query    string
	Category string // "services", "methods", "types", "enums", or empty for all
	Service  string // filter by specific service
	Limit    int
}

// Search performs a search across the SDK documentation index.
func (idx *SDKDocsIndex) Search(opts SearchOptions) []SearchResult {
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Limit > 50 {
		opts.Limit = 50
	}

	query := strings.ToLower(strings.TrimSpace(opts.Query))
	if query == "" {
		return nil
	}

	terms := tokenize(query)
	var results []SearchResult

	// Search services
	if opts.Category == "" || opts.Category == "services" {
		for name, service := range idx.Services {
			if opts.Service != "" && name != opts.Service {
				continue
			}
			score := computeScore(terms, name, service.Name, service.Description)
			if score > 0 {
				results = append(results, SearchResult{
					Type:        "service",
					Name:        service.Name,
					Path:        name,
					Description: truncate(service.Description, 200),
					Score:       score,
				})
			}
		}
	}

	// Search methods
	if opts.Category == "" || opts.Category == "methods" {
		for serviceName, service := range idx.Services {
			if opts.Service != "" && serviceName != opts.Service {
				continue
			}
			for methodName, method := range service.Methods {
				score := computeScore(terms, methodName, method.Name, method.Description)
				// Boost score if query contains the service name
				if containsAny(query, serviceName, service.Name) {
					score *= 1.5
				}
				if score > 0 {
					results = append(results, SearchResult{
						Type:        "method",
						Name:        methodName,
						Path:        serviceName + "." + methodName,
						Service:     serviceName,
						Description: truncate(method.Description, 200),
						Score:       score,
					})
				}
			}
		}
	}

	// Search types
	if opts.Category == "" || opts.Category == "types" {
		for typePath, typeDoc := range idx.Types {
			if opts.Service != "" && !strings.HasPrefix(typePath, opts.Service+".") {
				continue
			}
			score := computeScore(terms, typeDoc.Name, typePath, typeDoc.Description)
			if score > 0 {
				results = append(results, SearchResult{
					Type:        "type",
					Name:        typeDoc.Name,
					Path:        typePath,
					Service:     typeDoc.Package,
					Description: truncate(typeDoc.Description, 200),
					Score:       score,
				})
			}
		}
	}

	// Search enums
	if opts.Category == "" || opts.Category == "enums" {
		for enumPath, enumDoc := range idx.Enums {
			if opts.Service != "" && !strings.HasPrefix(enumPath, opts.Service+".") {
				continue
			}
			// Include enum values in search
			valuesStr := strings.Join(enumDoc.Values, " ")
			score := computeScore(terms, enumDoc.Name, enumPath, enumDoc.Description+" "+valuesStr)
			if score > 0 {
				results = append(results, SearchResult{
					Type:        "enum",
					Name:        enumDoc.Name,
					Path:        enumPath,
					Service:     enumDoc.Package,
					Description: truncate(enumDoc.Description, 200),
					Score:       score,
				})
			}
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		// Secondary sort by name for stability
		return results[i].Name < results[j].Name
	})

	// Apply limit
	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results
}

// tokenize splits a query into searchable terms.
func tokenize(query string) []string {
	// Split on common separators
	query = strings.NewReplacer(
		"_", " ",
		"-", " ",
		".", " ",
		",", " ",
		"?", " ",
		"!", " ",
	).Replace(query)

	words := strings.Fields(query)
	terms := make([]string, 0, len(words))

	// Filter out common stop words
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "is": true, "are": true,
		"to": true, "for": true, "in": true, "on": true, "of": true,
		"how": true, "do": true, "i": true, "can": true, "what": true,
		"get": true, "use": true, "with": true, "from": true,
	}

	for _, word := range words {
		word = strings.ToLower(word)
		if len(word) >= 2 && !stopWords[word] {
			terms = append(terms, word)
		}
	}

	return terms
}

// computeScore calculates a relevance score for a document.
func computeScore(queryTerms []string, names ...string) float64 {
	if len(queryTerms) == 0 {
		return 0
	}

	// Combine all searchable text
	combined := strings.ToLower(strings.Join(names, " "))

	var totalScore float64
	matchedTerms := 0

	for _, term := range queryTerms {
		termScore := 0.0

		// Exact word match (highest score)
		if containsWord(combined, term) {
			termScore = 10.0
			matchedTerms++
		} else if strings.Contains(combined, term) {
			// Substring match (lower score)
			termScore = 5.0
			matchedTerms++
		} else {
			// Try prefix matching
			words := strings.Fields(combined)
			for _, word := range words {
				if strings.HasPrefix(word, term) {
					termScore = 3.0
					matchedTerms++
					break
				}
			}
		}

		// Boost if term appears in first name (usually the identifier)
		if len(names) > 0 && strings.Contains(strings.ToLower(names[0]), term) {
			termScore *= 1.5
		}

		totalScore += termScore
	}

	// Require at least one term to match
	if matchedTerms == 0 {
		return 0
	}

	// Normalize by number of query terms and boost by match ratio
	matchRatio := float64(matchedTerms) / float64(len(queryTerms))
	return totalScore * matchRatio
}

// containsWord checks if text contains word as a complete word.
func containsWord(text, word string) bool {
	words := strings.Fields(text)
	for _, w := range words {
		if w == word {
			return true
		}
	}
	return false
}

// containsAny checks if text contains any of the given substrings.
func containsAny(text string, substrs ...string) bool {
	text = strings.ToLower(text)
	for _, s := range substrs {
		if strings.Contains(text, strings.ToLower(s)) {
			return true
		}
	}
	return false
}

// truncate shortens a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Find last space before maxLen to avoid cutting words
	if idx := strings.LastIndex(s[:maxLen], " "); idx > maxLen/2 {
		return s[:idx] + "..."
	}
	return s[:maxLen-3] + "..."
}
