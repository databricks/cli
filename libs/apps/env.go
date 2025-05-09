package apps

import (
	"strconv"
)

type EnvVar struct {
	Name  string
	Value string
}

func (e *EnvVar) String() string {
	return e.Name + "=" + e.Value
}

func GetBaseEnvVars(config *Config) []EnvVar {
	envVars := []EnvVar{
		{Name: "PYTHONUNBUFFERED", Value: "1"},
		{Name: "DATABRICKS_APP_NAME", Value: config.AppName},
		{Name: "DATABRICKS_APP_URL", Value: config.AppURL},
		{Name: "DATABRICKS_WORKSPACE_ID", Value: strconv.FormatInt(config.WorkspaceId, 10)},
		{Name: "DATABRICKS_HOST", Value: config.WorkspaceHost},
		{Name: "DATABRICKS_APP_PORT", Value: strconv.Itoa(config.Port)},
		{Name: "GRADIO_SERVER_NAME", Value: config.ServerName},
		{Name: "GRADIO_SERVER_PORT", Value: strconv.Itoa(config.Port)},
		{Name: "GRADIO_ANALYTICS_ENABLED", Value: "false"},
		{Name: "STREAMLIT_SERVER_PORT", Value: strconv.Itoa(config.Port)},
		{Name: "STREAMLIT_SERVER_ADDRESS", Value: config.ServerName},
		{Name: "STREAMLIT_SERVER_ENABLE_XSRF_PROTECTION", Value: "false"},
		{Name: "STREAMLIT_SERVER_ENABLE_CORS", Value: "false"},
		{Name: "STREAMLIT_SERVER_HEADLESS", Value: "true"},
		{Name: "STREAMLIT_BROWSER_GATHER_USAGE_STATS", Value: "false"},
		{Name: "UVICORN_PORT", Value: strconv.Itoa(config.Port)},
		{Name: "UVICORN_HOST", Value: config.ServerName},
		{Name: "PORT", Value: strconv.Itoa(config.Port)},
		{Name: "FLASK_RUN_PORT", Value: strconv.Itoa(config.Port)},
		{Name: "FLASK_RUN_HOST", Value: config.ServerName},
	}

	return envVars
}
