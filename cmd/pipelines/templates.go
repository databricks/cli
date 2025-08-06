package pipelines

const pipelineUpdateTemplate = `Update for pipeline {{- if .Update.Config }} {{ .Update.Config.Name }}{{ end }} completed successfully.{{- if .Update.Config }} Pipeline ID: {{ .Update.Config.Id }}{{ end }}
{{- if and .Update.CreationTime .LastEventTime }}
Started at {{ .Update.CreationTime | pretty_UTC_date_from_millis }} and completed at {{ .LastEventTime }}.
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
