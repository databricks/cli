package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/serving"
)

func PromptMessage() ([]serving.ChatMessage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, "emu/docs/source/dev-tools/cli")
	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}

	messages := []string{`
You're helping write documentation for the Databricks CLI.
Forget everything you know about the Databricks CLI and start from scratch.
You'll be provided with all the information you need to write the documentation.
Example invocations must be wrapped in Markdown code blocks.

What follows is existing official documentation for the Databricks CLI.
`}

	ignore := []string{
		"completion-commands.md",
		"migrate.md",
		"bundle-commands.md",
	}
	for _, file := range files {
		if slices.Contains(ignore, filepath.Base(file)) {
			continue
		}

		contents, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		messages = append(messages, fmt.Sprintf("File %s:\n\n------------%s\n------------", file, string(contents)))
	}

	return []serving.ChatMessage{
		{
			Role:    serving.ChatMessageRoleSystem,
			Content: strings.Join(messages, "\n\n"),
		},
	}, nil
}
