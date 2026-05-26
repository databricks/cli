package terraform_dabs_map

import "strings"

type renameRule struct {
	parent  string // DABs parent path suffix to match; "" = top-level only
	renames map[string]string
}

// renameRules maps DABs resource group names to their field rename rules.
// Each rule maps a DABs field name to its Terraform equivalent at a given parent path.
// Parent matching is suffix-based: rule "tasks" matches any path ending with "tasks".
var renameRules = map[string][]renameRule{
	"jobs": {
		{"", map[string]string{
			"tasks":        "task",
			"job_clusters": "job_cluster",
			"parameters":   "parameter",
			"environments": "environment",
		}},
		{"git_source", map[string]string{
			"git_branch":   "branch",
			"git_commit":   "commit",
			"git_provider": "provider",
			"git_tag":      "tag",
			"git_url":      "url",
		}},
		{"tasks", map[string]string{"libraries": "library"}},
		{"tasks.for_each_task.task", map[string]string{"libraries": "library"}},
	},
	"pipelines": {
		{"", map[string]string{
			"libraries":     "library",
			"clusters":      "cluster",
			"notifications": "notification",
		}},
	},
}

// applyRenames applies rename rules for a given DABs group to a field path.
// Returns the transformed TF path and the indices of renamed segments.
func applyRenames(group, fieldPath string) (string, []int) {
	rules := renameRules[group]
	if len(rules) == 0 {
		return fieldPath, nil
	}

	parts := strings.Split(fieldPath, ".")
	result := make([]string, len(parts))
	var renamedIdx []int

	for i, part := range parts {
		parent := strings.Join(parts[:i], ".")
		renamed := false
		for _, rule := range rules {
			if newName, ok := rule.renames[part]; ok && matchParent(parent, rule.parent) {
				result[i] = newName
				renamedIdx = append(renamedIdx, i)
				renamed = true
				break
			}
		}
		if !renamed {
			result[i] = part
		}
	}
	return strings.Join(result, "."), renamedIdx
}

// matchParent checks whether actualParent ends with the rulePath suffix.
// An empty rulePath matches only an empty actualParent (top-level fields).
func matchParent(actualParent, ruleParent string) bool {
	if ruleParent == "" {
		return actualParent == ""
	}
	ruleParts := strings.Split(ruleParent, ".")
	var actualParts []string
	if actualParent != "" {
		actualParts = strings.Split(actualParent, ".")
	}
	if len(actualParts) < len(ruleParts) {
		return false
	}
	offset := len(actualParts) - len(ruleParts)
	for i, rp := range ruleParts {
		if rp != actualParts[offset+i] {
			return false
		}
	}
	return true
}
