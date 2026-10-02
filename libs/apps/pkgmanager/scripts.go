package pkgmanager

import "strings"

// rewriteScript replaces script invocations at command boundaries, preserving
// argument text and quoting instead of reserializing the shell command.
func rewriteScript(script string, m Manager, scripts map[string]any) string {
	var out strings.Builder
	last := 0
	commandStart := true
	for pos := 0; pos < len(script); {
		start, end := scriptToken(script, pos)
		if start == end {
			break
		}
		pos = end
		word := script[start:end]
		if isScriptSeparator(word[0]) {
			commandStart = true
			continue
		}
		if !commandStart {
			continue
		}
		if isScriptAssignment(word) {
			continue
		}
		commandStart = false
		switch word {
		case "npm", "pnpm", "yarn", "bun":
		default:
			continue
		}

		nextStart, nextEnd := scriptToken(script, end)
		command := script[nextStart:nextEnd]
		replacement := m.Name
		scriptNameEnd := nextEnd
		if command == "run" {
			argStart, argEnd := scriptToken(script, nextEnd)
			if argStart == argEnd || isScriptSeparator(script[argStart]) || script[argStart] == '#' {
				continue // A bare `run` lists scripts; it does not invoke one.
			}
			scriptNameEnd = argEnd
		} else {
			if _, ok := scripts[command].(string); !ok || isPackageManagerCommand(command) {
				continue
			}
			replacement += " run"
		}
		out.WriteString(script[last:start])
		out.WriteString(replacement)
		last = end

		// npm consumes --; pnpm forwards every argument after the script name.
		// https://docs.npmjs.com/cli/commands/npm-run-script and https://pnpm.io/cli/run
		switch {
		case word == "pnpm" && m.Name == "npm":
			argStart, argEnd := scriptToken(script, scriptNameEnd)
			if argStart != argEnd && !isScriptSeparator(script[argStart]) && script[argStart] != '#' {
				out.WriteString(script[last:scriptNameEnd])
				out.WriteString(" --")
				last = scriptNameEnd
			}
		case word == "npm" && m.Name == "pnpm":
			before, after := npmArgumentSeparator(script, scriptNameEnd)
			if before != after {
				out.WriteString(script[last:before])
				last = after
			}
		}
	}
	out.WriteString(script[last:])
	return out.String()
}

// npmArgumentSeparator locates the first -- and its leading whitespace.
func npmArgumentSeparator(script string, pos int) (before, after int) {
	for {
		start, end := scriptToken(script, pos)
		if start == end || isScriptSeparator(script[start]) || script[start] == '#' {
			return pos, pos
		}
		switch script[start:end] {
		case "--", `'--'`, `"--"`:
			return pos, end
		}
		pos = end
	}
}

// isScriptAssignment recognizes shell NAME=value prefixes, including quoted values.
func isScriptAssignment(word string) bool {
	name, _, ok := strings.Cut(word, "=")
	if !ok || name == "" {
		return false
	}
	for i, c := range name {
		if c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (i > 0 && c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}

// isPackageManagerCommand distinguishes native operations from script shorthand.
// Native operations have manager-specific semantics and cannot be translated as
// script calls. See https://pnpm.io/cli/run and https://docs.npmjs.com/cli/commands.
func isPackageManagerCommand(command string) bool {
	switch command {
	case "add", "audit", "bin", "cache", "ci", "config", "create", "dedupe", "dlx",
		"env", "exec", "fetch", "help", "import", "info", "init", "install", "link",
		"list", "login", "logout", "outdated", "pack", "patch", "patch-commit",
		"prune", "publish", "rebuild", "remove", "root", "run", "run-script",
		"setup", "store", "uninstall", "unlink", "update", "upgrade", "version", "view", "why":
		return true
	}
	return false
}

// isScriptSeparator recognizes command boundaries supported by the rewriter.
func isScriptSeparator(c byte) bool {
	return strings.ContainsRune("&|;()\n", rune(c))
}

// scriptToken returns source offsets so untouched arguments retain their quoting
// and whitespace. Quoted or escaped separators are part of a word, not a command boundary.
func scriptToken(script string, pos int) (start, end int) {
	for pos < len(script) && strings.ContainsRune(" \t\r", rune(script[pos])) {
		pos++
	}
	start = pos
	if pos == len(script) {
		return start, pos
	}
	if isScriptSeparator(script[pos]) {
		return start, pos + 1
	}
	if script[pos] == '#' {
		for pos < len(script) && script[pos] != '\n' {
			pos++
		}
		return start, pos
	}
	var quote byte
	for pos < len(script) {
		c := script[pos]
		switch {
		case c == '\\' && quote != '\'', c == '^' && quote == 0:
			pos = min(pos+2, len(script))
			continue
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"' || c == '`':
			quote = c
		case strings.ContainsRune(" \t\r", rune(c)) || isScriptSeparator(c):
			return start, pos
		}
		pos++
	}
	return start, pos
}
