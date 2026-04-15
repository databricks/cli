package tableview

import (
	"context"
	"sync"

	"github.com/spf13/cobra"
)

type tableConfigKeyType struct{}

// SetTableConfig stores a *TableConfig in context.
func SetTableConfig(ctx context.Context, cfg *TableConfig) context.Context {
	return context.WithValue(ctx, tableConfigKeyType{}, cfg)
}

// GetTableConfig retrieves the *TableConfig from context, or nil.
func GetTableConfig(ctx context.Context) *TableConfig {
	cfg, _ := ctx.Value(tableConfigKeyType{}).(*TableConfig)
	return cfg
}

// configByCmd holds configs registered via SetTableConfigOnCmd so they can
// be looked up before the command's PreRunE has executed (useful in tests).
var configByCmd sync.Map

// SetTableConfigOnCmd arranges for cfg to be stored in the command's context
// at execution time via PreRunE, preserving Cobra's context propagation.
func SetTableConfigOnCmd(cmd *cobra.Command, cfg *TableConfig) {
	configByCmd.Store(cmd, cfg)
	prev := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(SetTableConfig(cmd.Context(), cfg))
		if prev != nil {
			return prev(cmd, args)
		}
		return nil
	}
}

// GetTableConfigForCmd returns the config registered on cmd via
// SetTableConfigOnCmd. Unlike GetTableConfig (which reads from context),
// this works before the command's PreRunE has executed.
func GetTableConfigForCmd(cmd *cobra.Command) *TableConfig {
	v, ok := configByCmd.Load(cmd)
	if !ok {
		return nil
	}
	return v.(*TableConfig)
}
