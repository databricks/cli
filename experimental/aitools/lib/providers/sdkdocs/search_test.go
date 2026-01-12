package sdkdocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple query",
			input:    "create job",
			expected: []string{"create", "job"},
		},
		{
			name:     "query with stop words",
			input:    "how do I create a job",
			expected: []string{"create", "job"},
		},
		{
			name:     "query with underscores",
			input:    "cluster_name field",
			expected: []string{"cluster", "name", "field"},
		},
		{
			name:     "empty query",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only stop words",
			input:    "how do I",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeScore(t *testing.T) {
	tests := []struct {
		name       string
		queryTerms []string
		names      []string
		expectZero bool
	}{
		{
			name:       "exact match",
			queryTerms: []string{"create"},
			names:      []string{"Create", "Create a new job"},
			expectZero: false,
		},
		{
			name:       "no match",
			queryTerms: []string{"delete"},
			names:      []string{"Create", "Create a new job"},
			expectZero: true,
		},
		{
			name:       "partial match",
			queryTerms: []string{"job"},
			names:      []string{"CreateJob", "Creates a job"},
			expectZero: false,
		},
		{
			name:       "empty query",
			queryTerms: []string{},
			names:      []string{"Create", "Create a new job"},
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeScore(tt.queryTerms, tt.names...)
			if tt.expectZero {
				assert.Equal(t, float64(0), score)
			} else {
				assert.Greater(t, score, float64(0))
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string truncated at word boundary",
			input:    "hello world this is a long string",
			maxLen:   15,
			expected: "hello world...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSearch(t *testing.T) {
	// Create a test index
	index := &SDKDocsIndex{
		Version: "1.0",
		Services: map[string]*ServiceDoc{
			"jobs": {
				Name:        "Jobs",
				Description: "The Jobs API allows you to create, edit, and delete jobs.",
				Package:     "github.com/databricks/databricks-sdk-go/service/jobs",
				Methods: map[string]*MethodDoc{
					"Create": {
						Name:        "Create",
						Description: "Create a new job.",
						Signature:   "Create(ctx context.Context, request CreateJob) (*CreateResponse, error)",
					},
					"List": {
						Name:        "List",
						Description: "List all jobs.",
						Signature:   "List(ctx context.Context, request ListJobsRequest) listing.Iterator[BaseJob]",
					},
					"Delete": {
						Name:        "Delete",
						Description: "Delete a job.",
						Signature:   "Delete(ctx context.Context, request DeleteJob) error",
					},
				},
			},
			"compute": {
				Name:        "Clusters",
				Description: "The Clusters API allows you to create and manage clusters.",
				Package:     "github.com/databricks/databricks-sdk-go/service/compute",
				Methods: map[string]*MethodDoc{
					"Create": {
						Name:        "Create",
						Description: "Create a new cluster.",
						Signature:   "Create(ctx context.Context, request CreateCluster) (*CreateClusterResponse, error)",
					},
				},
			},
		},
		Types: map[string]*TypeDoc{
			"jobs.CreateJob": {
				Name:        "CreateJob",
				Package:     "jobs",
				Description: "Job creation settings.",
				Fields: map[string]*FieldDoc{
					"name": {
						Name:        "name",
						Type:        "string",
						Description: "The job name.",
					},
				},
			},
		},
		Enums: map[string]*EnumDoc{
			"jobs.RunLifeCycleState": {
				Name:        "RunLifeCycleState",
				Package:     "jobs",
				Description: "The current state of the run lifecycle.",
				Values:      []string{"PENDING", "RUNNING", "TERMINATED"},
			},
		},
	}

	t.Run("search for create job", func(t *testing.T) {
		results := index.Search(SearchOptions{
			Query: "create job",
			Limit: 10,
		})

		require.NotEmpty(t, results)
		// Should find the Jobs.Create method
		found := false
		for _, r := range results {
			if r.Type == "method" && r.Name == "Create" && r.Service == "jobs" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find Jobs.Create method")
	})

	t.Run("search with service filter", func(t *testing.T) {
		results := index.Search(SearchOptions{
			Query:   "create",
			Service: "jobs",
			Limit:   10,
		})

		for _, r := range results {
			if r.Type == "method" {
				assert.Equal(t, "jobs", r.Service, "All method results should be from jobs service")
			}
		}
	})

	t.Run("search with category filter", func(t *testing.T) {
		results := index.Search(SearchOptions{
			Query:    "job",
			Category: "types",
			Limit:    10,
		})

		for _, r := range results {
			assert.Equal(t, "type", r.Type, "All results should be types")
		}
	})

	t.Run("search for enum values", func(t *testing.T) {
		results := index.Search(SearchOptions{
			Query:    "lifecycle state",
			Category: "enums",
			Limit:    10,
		})

		require.NotEmpty(t, results)
		found := false
		for _, r := range results {
			if r.Name == "RunLifeCycleState" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find RunLifeCycleState enum")
	})

	t.Run("empty query returns no results", func(t *testing.T) {
		results := index.Search(SearchOptions{
			Query: "",
			Limit: 10,
		})

		assert.Empty(t, results)
	})

	t.Run("limit is enforced", func(t *testing.T) {
		results := index.Search(SearchOptions{
			Query: "create",
			Limit: 1,
		})

		assert.LessOrEqual(t, len(results), 1)
	})
}
