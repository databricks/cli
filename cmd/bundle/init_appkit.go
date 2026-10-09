package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

type appKitInitOptions struct {
	configFile  string
	outputDir   string
	templateDir string
	tag         string
	branch      string
}

type appKitInitConfig struct {
	ProjectName    string   `json:"project_name"`
	AppDescription string   `json:"app_description"`
	Features       []string `json:"features"`
	Set            []string `json:"set"`
	AuthMode       string   `json:"auth_mode"`
	PackageManager string   `json:"package_manager"`
	SkipInstall    bool     `json:"skip_install"`
	Version        string   `json:"version"`
}

func readAppKitInitConfig(path string) (appKitInitConfig, error) {
	var config appKitInitConfig
	if path == "" {
		return config, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return config, fmt.Errorf("open AppKit config file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("read AppKit config file: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return config, errors.New("AppKit config file must contain one JSON object")
	}
	return config, nil
}

func runAppKitInit(cmd *cobra.Command, newAppKitInit func() *cobra.Command, opts appKitInitOptions) error {
	if opts.templateDir != "" {
		return errors.New("--template-dir is not supported for app-appkit; use databricks apps init --template for a custom AppKit template")
	}
	if root.OutputType(cmd) == flags.OutputJSON {
		return errors.New("-o json is not supported for app-appkit; use text output")
	}

	config, err := readAppKitInitConfig(opts.configFile)
	if err != nil {
		return err
	}
	if config.Version != "" && (opts.tag != "" || opts.branch != "") {
		return errors.New("version in --config-file cannot be combined with --tag or --branch")
	}
	if config.ProjectName == "" && !cmdio.IsPromptSupported(cmd.Context()) {
		return errors.New("app-appkit requires project_name in --config-file in non-interactive mode")
	}

	appCmd := newAppKitInit()
	appCmd.SetContext(cmd.Context())
	appFlags := appCmd.Flags()
	values := map[string]string{
		"name":            config.ProjectName,
		"description":     config.AppDescription,
		"output-dir":      opts.outputDir,
		"auth-mode":       config.AuthMode,
		"package-manager": config.PackageManager,
	}
	for flag, value := range values {
		if value == "" {
			continue
		}
		if err := appFlags.Set(flag, value); err != nil {
			return fmt.Errorf("set AppKit %s: %w", flag, err)
		}
	}
	if len(config.Features) > 0 {
		if err := appFlags.Set("features", strings.Join(config.Features, ",")); err != nil {
			return fmt.Errorf("set AppKit features: %w", err)
		}
	}
	for _, value := range config.Set {
		if err := appFlags.Set("set", value); err != nil {
			return fmt.Errorf("set AppKit resource value: %w", err)
		}
	}
	if opts.branch != "" || opts.tag != "" {
		ref := opts.branch
		if ref == "" {
			ref = opts.tag
		}
		if err := appFlags.Set("branch", ref); err != nil {
			return fmt.Errorf("set AppKit template ref: %w", err)
		}
	} else if config.Version != "" {
		if err := appFlags.Set("version", config.Version); err != nil {
			return fmt.Errorf("set AppKit version: %w", err)
		}
	}
	if config.SkipInstall {
		if err := appFlags.Set("skip-install", "true"); err != nil {
			return fmt.Errorf("set AppKit skip-install: %w", err)
		}
	}

	return appCmd.RunE(appCmd, nil)
}
