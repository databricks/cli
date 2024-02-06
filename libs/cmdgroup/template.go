package cmdgroup

const usageTemplate = `Usage:{{if .Command.Runnable}}
  {{.Command.UseLine}}{{end}}
{{range .FlagGroups}}
{{.Name}} Flags:{{if not (eq .Description "")}}
  {{.Description}}{{end}}
{{.FlagSet.FlagUsages | trimTrailingWhitespaces}}
{{end}}
{{if .Command.HasAvailableLocalFlags}}Flags:
{{.Command.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .Command.HasAvailableInheritedFlags}}

Global Flags:
{{.Command.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}`
