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
}

const (
	DEFAULT_APP_NAME = "app"
	DEFAULT_HOST     = "127.0.0.1"
	DEFAULT_PORT     = 8000
)

func NewConfig(workspaceHost string, workpaceId int64, appDir string) *Config {
	c := &Config{
		AppName:       DEFAULT_APP_NAME,
		AppURL:        fmt.Sprintf("http://%s:%d", DEFAULT_HOST, DEFAULT_PORT),
		WorkspaceId:   workpaceId,
		ServerName:    DEFAULT_HOST,
		Port:          DEFAULT_PORT,
		Host:          DEFAULT_HOST,
		WorkspaceHost: workspaceHost,
		AppPath:       appDir,
		AppSpecFiles:  []string{"app.yml", "app.yaml"},
	}

	return c
}
