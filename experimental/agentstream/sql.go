package agentstream

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SQL tool name constants.
const (
	toolExecuteSQL      = "execute_sql"
	toolExecuteSQLQuery = "execute_sql_query"
)

// sqlArgs is the JSON shape of execute_sql function call arguments.
// execute_sql uses {"sql": "...", "title": "..."}.
type sqlArgs struct {
	SQL   string `json:"sql"`
	Title string `json:"title"`
}

// sqlQueryArgs is the JSON shape of execute_sql_query function call arguments.
// execute_sql_query uses {"query": "...", "thought": "..."}.
type sqlQueryArgs struct {
	Query   string `json:"query"`
	Thought string `json:"thought"`
}

// isSQLTool returns true if the tool name is a SQL execution tool.
func isSQLTool(name string) bool {
	return name == toolExecuteSQL || name == toolExecuteSQLQuery
}

// renderSQL parses SQL tool arguments and prints the SQL query.
func renderSQL(w io.Writer, name, arguments string) {
	sql, title := parseSQLArgs(name, arguments)
	if sql == "" {
		return
	}
	fmt.Fprintln(w)
	if title != "" {
		fmt.Fprintf(w, "SQL executed (%s):\n", title)
	} else {
		fmt.Fprintln(w, "SQL executed:")
	}
	for _, line := range strings.Split(sql, "\n") {
		fmt.Fprintf(w, "  %s\n", line)
	}
}

// parseSQLArgs extracts SQL and title from SQL tool arguments.
// Handles both execute_sql and execute_sql_query argument formats.
func parseSQLArgs(name, arguments string) (sql, title string) {
	switch name {
	case toolExecuteSQL:
		var args sqlArgs
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", ""
		}
		return args.SQL, args.Title
	case toolExecuteSQLQuery:
		var args sqlQueryArgs
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", ""
		}
		return args.Query, ""
	default:
		return "", ""
	}
}
