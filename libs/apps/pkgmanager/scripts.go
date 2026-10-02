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
		if word == "env" {
			pos = envCommandStart(script, end)
			continue
		}
		commandStart = false
		switch word {
		case "npm", "pnpm", "yarn", "bun":
		default:
			continue
		}

		var options strings.Builder
		beforeCommand := end
		for optionEnd := scriptManagerOptionEnd(script, beforeCommand); optionEnd != beforeCommand; optionEnd = scriptManagerOptionEnd(script, beforeCommand) {
			options.WriteString(script[beforeCommand:optionEnd])
			beforeCommand = optionEnd
		}
		nextStart, nextEnd := scriptToken(script, beforeCommand)
		command := script[nextStart:nextEnd]
		run := " run"
		beforeName, nameEnd := beforeCommand, nextEnd
		if command == "run" || command == "run-script" {
			run = script[beforeCommand:nextEnd]
			beforeName = nextEnd
			for optionEnd := scriptManagerOptionEnd(script, beforeName); optionEnd != beforeName; optionEnd = scriptManagerOptionEnd(script, beforeName) {
				options.WriteString(script[beforeName:optionEnd])
				beforeName = optionEnd
			}
			argStart, argEnd := scriptToken(script, beforeName)
			if argStart == argEnd || isScriptSeparator(script[argStart]) || script[argStart] == '#' {
				continue // A bare `run` lists scripts; it does not invoke one.
			}
			nameEnd = argEnd
			if word == m.Name {
				continue
			}
		} else if _, ok := scripts[command].(string); !ok || isPackageManagerCommand(command) {
			continue
		}

		// npm consumes --; pnpm forwards every argument after the script name.
		// https://docs.npmjs.com/cli/commands/npm-run-script and https://pnpm.io/cli/run
		args := ""
		pos = nameEnd
		switch {
		case word == "pnpm" && m.Name == "npm":
			argStart, argEnd := scriptToken(script, nameEnd)
			if argStart != argEnd && !isScriptSeparator(script[argStart]) && script[argStart] != '#' {
				args = " --"
			}
		case word == "npm" && m.Name == "pnpm":
			var trailingOptions string
			trailingOptions, args, pos = npmScriptArguments(script, nameEnd)
			options.WriteString(trailingOptions)
		}
		out.WriteString(script[last:start])
		out.WriteString(m.Name)
		out.WriteString(run)
		out.WriteString(options.String())
		out.WriteString(script[beforeName:nameEnd])
		out.WriteString(args)
		last = pos
	}
	out.WriteString(script[last:])
	return out.String()
}

// envCommandStart skips env options, leaving assignments for the command scanner.
// https://pubs.opengroup.org/onlinepubs/9799919799/utilities/env.html
func envCommandStart(script string, pos int) int {
	for {
		start, end := scriptToken(script, pos)
		word := script[start:end]
		switch {
		case word == "--":
			return end
		case word == "-i" || word == "--ignore-environment" || strings.HasPrefix(word, "--unset="):
			pos = end
		case word == "-u" || word == "--unset":
			_, pos = scriptToken(script, end)
		default:
			return pos
		}
	}
}

// scriptManagerOptionEnd includes a separate value for options that take one.
// Boolean options must not consume positional script arguments.
func scriptManagerOptionEnd(script string, pos int) int {
	start, end := scriptToken(script, pos)
	word := scriptWordValue(script[start:end])
	if !strings.HasPrefix(word, "-") || word == "--" {
		return pos
	}
	valueStart, valueEnd := scriptToken(script, end)
	if valueStart == valueEnd || isScriptSeparator(script[valueStart]) || script[valueStart] == '#' {
		return end
	}
	switch word {
	case "--loglevel", "--script-shell", "--prefix", "--dir", "-C", "--workspace", "-w", "--filter", "-F":
		return valueEnd
	case "--if-present", "--no-if-present":
		// Logging shortcuts such as --silent and -s do not take boolean values.
		switch scriptWordValue(script[valueStart:valueEnd]) {
		case "true", "false":
			return valueEnd
		}
	}
	return end
}

// npmScriptArguments separates npm options from child arguments up to a command boundary.
// Unlike pnpm, npm consumes manager options anywhere before --, even after the script name.
func npmScriptArguments(script string, pos int) (options, args string, end int) {
	var managerOptions, scriptArgs strings.Builder
	argumentsOnly := false
	for {
		start, end := scriptToken(script, pos)
		if start == end || isScriptSeparator(script[start]) || script[start] == '#' {
			return managerOptions.String(), scriptArgs.String(), pos
		}
		if !argumentsOnly {
			if scriptWordValue(script[start:end]) == "--" {
				argumentsOnly = true
				pos = end
				continue
			}
			if optionEnd := scriptManagerOptionEnd(script, pos); optionEnd != pos {
				managerOptions.WriteString(script[pos:optionEnd])
				pos = optionEnd
				continue
			}
		}
		scriptArgs.WriteString(script[pos:end])
		pos = end
	}
}

// scriptWordValue removes shell quoting for classification without expanding variables.
// The rewriter emits the original source text, preserving literal quotes and escapes.
// https://pubs.opengroup.org/onlinepubs/9799919799/utilities/V3_chap02.html#tag_19_02
func scriptWordValue(word string) string {
	var value strings.Builder
	var quote byte
	for pos := 0; pos < len(word); pos++ {
		c := word[pos]
		switch {
		case c == '\\' && quote != '\'':
			// Within double quotes, backslashes only escape these shell metacharacters.
			if pos+1 < len(word) && (quote == 0 || strings.ContainsRune("$`\"\\\n", rune(word[pos+1]))) {
				pos++
				if word[pos] != '\n' {
					value.WriteByte(word[pos])
				}
			} else {
				value.WriteByte(c)
			}
		case quote != 0 && c == quote:
			quote = 0
		case quote == 0 && (c == '\'' || c == '"'):
			quote = c
		default:
			value.WriteByte(c)
		}
	}
	return value.String()
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
