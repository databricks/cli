package agentstream

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// sqlArgs is the JSON shape of execute_sql function call arguments.
type sqlArgs struct {
	SQL   string `json:"sql"`
	Title string `json:"title"`
}

// renderSQL parses execute_sql arguments and prints the SQL query.
func renderSQL(w io.Writer, arguments string) {
	var args sqlArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil || args.SQL == "" {
		return
	}
	fmt.Fprintln(w)
	if args.Title != "" {
		fmt.Fprintf(w, "SQL executed (%s):\n", args.Title)
	} else {
		fmt.Fprintln(w, "SQL executed:")
	}
	for _, line := range strings.Split(args.SQL, "\n") {
		fmt.Fprintf(w, "  %s\n", line)
	}
}

// parseSQLArgs extracts SQL and title from execute_sql function call arguments.
func parseSQLArgs(arguments string) (sql, title string) {
	var args sqlArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", ""
	}
	return args.SQL, args.Title
}
