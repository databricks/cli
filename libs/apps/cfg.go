package apps

import "fmt"

type Config struct {
	AppName       string
	AppURL        string
	WorkspaceId   int64
	ServerName    string
	Host          string
	WorkspaceHost string
	Port          int
	AppPath       string
	AppSpecFiles  []string
	DebugPort     string
}

const (
	DEFAULT_APP_NAME = "app"
	DEFAULT_HOST     = "127.0.0.1"
	DEFAULT_PORT     = 8000
)

func NewConfig(workspaceHost string, workpaceId int64, appDir, host string, port int) *Config {
	c := &Config{
		AppName:       DEFAULT_APP_NAME,
		AppURL:        fmt.Sprintf("http://%s:%d", host, port),
		WorkspaceId:   workpaceId,
		ServerName:    host,
		Port:          port,
		Host:          host,
		WorkspaceHost: workspaceHost,
		AppPath:       appDir,
		AppSpecFiles:  []string{"app.yml", "app.yaml"},
	}

	return c
}
