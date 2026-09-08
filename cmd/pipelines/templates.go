package pipelines

const pipelineHistoryTemplate = `Updates Summary for pipeline {{.Key}}:
{{if .Updates}}{{range .Updates}}Update ID: {{.UpdateId}}
{{- if .State }}
   State: {{.State}}
{{- end}}
{{- if .Cause }}
   Cause: {{.Cause}}
{{- end}}
{{- if .CreationTime }}
   Creation Time: {{.CreationTime | pretty_UTC_date_from_millis}}
{{- end}}
   Full Refresh: {{.FullRefresh}}
   Validate Only: {{.ValidateOnly}}
{{end}}{{else}}No updates found.{{end}}`

const pipelineUpdateTemplate = `
Update for pipeline {{- if .Update.Config }} {{ .Update.Config.Name }}{{ end }} completed successfully.
{{- if .Update.Config }}

Pipeline ID: {{ .Update.Config.Id }}
{{- end }}
{{- if and .Update.CreationTime .LastEventTime }}
Update start time: {{ .Update.CreationTime | pretty_UTC_date_from_millis }}
Update end time: {{ .LastEventTime }}
{{- end }}

Pipeline configurations for this update:
{{- if .Update.FullRefresh }}
• All tables are fully refreshed
{{- else if and (eq (len .Update.RefreshSelection) 0) (eq (len .Update.FullRefreshSelection) 0) }}
• All tables are refreshed
{{- else }}
{{- if gt (len .Update.RefreshSelection) 0 }}
• Refreshed [{{ join .Update.RefreshSelection ", " }}]
{{- end }}
{{- if gt (len .Update.FullRefreshSelection) 0 }}
• Full refreshed [{{ join .Update.FullRefreshSelection ", " }}]
{{- end }}
{{- end }}
{{- if .Update.Cause }}
• Update cause: {{ .Update.Cause }}
{{- end }}
{{- if .Update.Config }}
{{- if .Update.Config.Serverless }}
• Serverless compute
{{- else if .Update.ClusterId }}
• Classic compute: {{ .Update.ClusterId }}
{{- end }}
{{- else if .Update.ClusterId }}
• Classic compute: {{ .Update.ClusterId }}
{{- end }}
{{- if .Update.Config }}
{{- if .Update.Config.Channel }}
• Channel: {{ .Update.Config.Channel }}
{{- end }}
{{- if .Update.Config.Continuous }}
• {{ if .Update.Config.Continuous }}Continuous{{ else }}Triggered{{ end }} pipeline
{{- end }}
{{- if .Update.Config.Development }}
• {{ if .Update.Config.Development }}Development{{ else }}Production{{ end }} mode
{{- end }}
{{- if .Update.Config.Catalog }}
• Catalog: {{ .Update.Config.Catalog }}
{{- if .Update.Config.Schema }}
• Schema: {{ .Update.Config.Schema }}
{{- end }}
{{- else if .Update.Config.Storage }}
• Storage: {{ .Update.Config.Storage }}
{{- end }}
{{- end }}
`

// pipelineDescribeTemplate renders the `pipelines describe` summary: pipeline
// configuration and state, followed by the most recent update (if any).
const pipelineDescribeTemplate = `Pipeline: {{if .Pipeline.Name}}{{.Pipeline.Name}}{{else}}{{.Pipeline.PipelineId}}{{end}}
{{- if .Key}}
Key: {{.Key}}
{{- end}}
ID: {{.Pipeline.PipelineId}}
{{- if .Pipeline.State}}
State: {{.Pipeline.State}}
{{- end}}
{{- if .Pipeline.Health}}
Health: {{.Pipeline.Health}}
{{- end}}
{{- if .Pipeline.CreatorUserName}}
Creator: {{.Pipeline.CreatorUserName}}
{{- end}}
{{- if .Pipeline.RunAsUserName}}
Run as: {{.Pipeline.RunAsUserName}}
{{- end}}
{{- with .Pipeline.Spec}}
{{- if .Catalog}}
Target: {{.Catalog}}{{if .Schema}}.{{.Schema}}{{end}}
{{- end}}
Mode: {{if .Continuous}}Continuous{{else}}Triggered{{end}}, {{if .Development}}Development{{else}}Production{{end}}
Compute: {{if .Serverless}}Serverless{{else if $.Pipeline.ClusterId}}Classic ({{$.Pipeline.ClusterId}}){{else}}Classic{{end}}
{{- if .Channel}}
Channel: {{.Channel}}
{{- end}}
{{- end}}

Last run:
{{- if .LastUpdate}}
  Update ID: {{.LastUpdate.UpdateId}}
{{- if .LastUpdate.State}}
  State: {{.LastUpdate.State}}
{{- end}}
{{- if .LastUpdate.CreationTime}}
  Started: {{.LastUpdate.CreationTime | pretty_UTC_date_from_millis}}
{{- end}}
{{- if .LastUpdate.FullRefresh}}
  Full refresh: all tables
{{- end}}
{{- if .LastUpdate.RefreshSelection}}
  Refreshed: [{{join .LastUpdate.RefreshSelection ", "}}]
{{- end}}
{{- if .LastUpdate.FullRefreshSelection}}
  Full refreshed: [{{join .LastUpdate.FullRefreshSelection ", "}}]
{{- end}}
{{- if .LastUpdate.Cause}}
  Cause: {{.LastUpdate.Cause}}
{{- end}}
{{- else}}
  No runs yet.
{{- end}}
`

// progressEventsTemplate is the template for displaying progress events
const progressEventsTemplate = `{{- if .ProgressEvents }}
{{ printf "%-25s %s\n" "Run Phase" "Duration" }}
{{- printf "%-25s %s\n" "---------" "--------" }}
{{- range .ProgressEvents }}
{{- printf "%-25s %s\n" .Phase .Duration }}
{{- end }}
{{- end }}`
