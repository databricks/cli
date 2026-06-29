package aitools

import (
	"context"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/spf13/cobra"
)

// Package-level for testability. Tests override via uninstall_test.go.
var uninstallSkillsFn = func(ctx context.Context, opts installer.UninstallOptions) error {
	return installer.UninstallSkillsOpts(ctx, opts)
}

func NewUninstallCmd() *cobra.Command {
	var skillsFlag, scopeFlag string
	var projectFlag, globalFlag, keepMarketplace bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall AI skills",
		Long: `Remove installed Databricks AI skills from all coding agents.

By default, removes all skills. Use --skills to remove specific skills only.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			projectFlag, globalFlag, err := parseScopeFlag(scopeFlag, projectFlag, globalFlag, false)
			if err != nil {
				return err
			}

			globalDir, err := installer.GlobalSkillsDir(ctx)
			if err != nil {
				return err
			}
			projectDir, err := installer.ProjectSkillsDir(ctx)
			if err != nil {
				return err
			}

			scope, err := resolveScopeForUninstall(ctx, projectFlag, globalFlag, globalDir, projectDir)
			if err != nil {
				return err
			}

			opts := installer.UninstallOptions{
				Scope:           scope,
				KeepMarketplace: keepMarketplace,
			}
			opts.Skills = splitAndTrim(skillsFlag)
			return uninstallSkillsFn(ctx, opts)
		},
	}

	cmd.Flags().StringVar(&skillsFlag, "skills", "", "Specific skills to uninstall (comma-separated)")
	cmd.Flags().BoolVar(&keepMarketplace, "keep-marketplace", false, "Keep the marketplace registration when removing a plugin")
	cmd.Flags().StringVar(&scopeFlag, "scope", "", "Uninstall scope: project or global")
	cmd.Flags().BoolVar(&projectFlag, "project", false, "Uninstall project-scoped skills")
	cmd.Flags().BoolVar(&globalFlag, "global", false, "Uninstall globally-scoped skills")
	markScopeBoolsDeprecated(cmd)
	return cmd
}
