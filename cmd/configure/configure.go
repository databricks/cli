package configure

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/cmd/prompt"
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

type Configs struct {
	Host  string `ini:"host"`
	Token string `ini:"token"`
}

var noInteractive bool

func (cfg *Configs) readFromStdin() error {
	n, err := fmt.Scanf("%s %s\n", &cfg.Host, &cfg.Token)
	if err != nil {
		return err
	}
	if n != 2 {
		return fmt.Errorf("exactly 2 arguments are required")
	}
	return nil
}

func (cfg *Configs) prompt() error {
	res := prompt.Results{}
	err := prompt.Questions{prompt.Text{
		Key:   "host",
		Label: "Databricks Host",
		Default: func(res prompt.Results) string {
			return cfg.Host
		},
		Callback: func(ans prompt.Answer, prj *project.Project, res prompt.Results) {
			cfg.Host = ans.Value
		},
	}, prompt.Text{
		Key:   "token",
		Label: "Databricks Token",
		Default: func(res prompt.Results) string {
			return cfg.Token
		},
		Callback: func(ans prompt.Answer, prj *project.Project, res prompt.Results) {
			cfg.Token = ans.Value
		},
	}}.Ask(res)
	if err != nil {
		return err
	}

	for _, answer := range res {
		if answer.Callback != nil {
			answer.Callback(answer, nil, res)
		}
	}

	return nil
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure authentication",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		path := os.Getenv("DATABRICKS_CONFIG_FILE")
		if path == "" {
			path, err = os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("homedir: %w", err)
			}
		}
		if filepath.Base(path) == ".databrickscfg" {
			path = filepath.Dir(path)
		}
		err = os.MkdirAll(path, os.ModeDir|os.ModePerm)
		if err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
		cfgPath := filepath.Join(path, ".databrickscfg")
		_, err = os.Stat(cfgPath)
		if errors.Is(err, os.ErrNotExist) {
			file, err := os.Create(cfgPath)
			if err != nil {
				return fmt.Errorf("create config file: %w", err)
			}
			file.Close()
		} else if err != nil {
			return fmt.Errorf("open config file: %w", err)
		}

		ini_cfg, err := ini.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("load config file: %w", err)
		}
		cfg := &Configs{"", ""}
		err = ini_cfg.MapTo(cfg)
		if err != nil {
			return fmt.Errorf("unmarshal loaded config: %w", err)
		}

		if noInteractive {
			err = cfg.readFromStdin()
		} else {
			err = cfg.prompt()
		}
		if err != nil {
			return fmt.Errorf("reading configs: %w", err)
		}

		var buffer bytes.Buffer
		buffer.WriteString("[DEFAULT]\n")
		err = ini_cfg.ReflectFrom(cfg)
		if err != nil {
			return fmt.Errorf("marshall config: %w", err)
		}
		_, err = ini_cfg.WriteTo(&buffer)
		if err != nil {
			return fmt.Errorf("write config to buffer: %w", err)
		}
		err = os.WriteFile(cfgPath, buffer.Bytes(), os.ModePerm)
		if err != nil {
			return fmt.Errorf("write congfig to file: %w", err)
		}

		return nil
	},
}

func init() {
	root.RootCmd.AddCommand(configureCmd)
	configureCmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "Don't show interactive prompts for inputs. Read directly from stdin")
}
