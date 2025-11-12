# Step 8: CLI and Deployment

## Overview
Finalize CLI, create installation scripts, add version checking, and write documentation for deployment and usage.

## Tasks

### 8.1 Enhance CLI with Subcommands

**cmd/go-mcp/cli.go:**

```go
type CLI struct {
    rootCmd *cobra.Command
}

func NewCLI() *CLI {
    cli := &CLI{}

    rootCmd := &cobra.Command{
        Use:   "go-mcp",
        Short: "Go MCP Server for Databricks integration",
        Long: `A Model Context Protocol (MCP) server providing Databricks
integration, project scaffolding, and workspace tools.`,
        RunE: cli.runServer,
    }

    // Global flags
    rootCmd.PersistentFlags().String("config", "", "Config file path (default: ~/.go-mcp/config.json)")
    rootCmd.PersistentFlags().Bool("disallow-deployment", false, "Disable deployment operations")
    rootCmd.PersistentFlags().Bool("with-workspace-tools", false, "Enable workspace tools")

    // Subcommands
    rootCmd.AddCommand(
        cli.versionCmd(),
        cli.checkCmd(),
        cli.configCmd(),
    )

    cli.rootCmd = rootCmd
    return cli
}

func (cli *CLI) Execute() error {
    return cli.rootCmd.Execute()
}

func (cli *CLI) versionCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "version",
        Short: "Print version information",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Printf("go-mcp version %s\n", version.Version)
            fmt.Printf("  Commit:     %s\n", version.Commit)
            fmt.Printf("  Build time: %s\n", version.BuildTime)
        },
    }
}

func (cli *CLI) checkCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "check",
        Short: "Check environment configuration",
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg, err := config.LoadConfig()
            if err != nil {
                return err
            }
            return checkEnvironment(cfg, slog.Default())
        },
    }
}

func (cli *CLI) configCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "config",
        Short: "Manage configuration",
    }

    cmd.AddCommand(
        &cobra.Command{
            Use:   "show",
            Short: "Show current configuration",
            RunE:  cli.showConfig,
        },
        &cobra.Command{
            Use:   "init",
            Short: "Initialize default configuration",
            RunE:  cli.initConfig,
        },
    )

    return cmd
}

func (cli *CLI) showConfig(cmd *cobra.Command, args []string) error {
    cfg, err := config.LoadConfig()
    if err != nil {
        return err
    }

    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }

    fmt.Println(string(data))
    return nil
}

func (cli *CLI) initConfig(cmd *cobra.Command, args []string) error {
    cfg := config.DefaultConfig()
    return cfg.Save()
}

func (cli *CLI) runServer(cmd *cobra.Command, args []string) error {
    // Load and validate config
    cfg, err := config.LoadConfig()
    if err != nil {
        return err
    }

    // Apply flags
    if disallow, _ := cmd.Flags().GetBool("disallow-deployment"); disallow {
        cfg.AllowDeployment = false
    }
    if withWorkspace, _ := cmd.Flags().GetBool("with-workspace-tools"); withWorkspace {
        cfg.WithWorkspaceTools = true
    }

    if err := cfg.Validate(); err != nil {
        return err
    }

    // Create logger
    sessionID := generateSessionID()
    logger := logging.NewLogger(sessionID, true)

    // Version check in background
    go version.CheckForUpdates(cmd.Context())

    // Print banner
    printBanner(sessionID, logger)

    // Create and run server
    handler, err := server.NewCombinedHandler(cfg, logger)
    if err != nil {
        return err
    }

    srv := server.NewServer(handler, logger)
    return srv.Run(cmd.Context())
}
```

## Next Steps

After completing all 8 steps:
1. Tag first release (v0.1.0)
2. Test with real MCP client (Claude Code)
3. Gather feedback and iterate
4. Add Dagger sandbox backend
5. Extend with additional providers as needed
