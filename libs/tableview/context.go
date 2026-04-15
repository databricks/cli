package tableview

import "context"

type tableConfigKeyType struct{}

// SetTableConfig stores a *TableConfig in context.
// If ctx is nil (e.g. during command construction before Execute), context.Background() is used.
func SetTableConfig(ctx context.Context, cfg *TableConfig) context.Context {
	if ctx == nil {
		// Commands have no context during construction (before Execute), so
		// context.Background is the only available root.
		ctx = context.Background() //nolint:gocritic
	}
	return context.WithValue(ctx, tableConfigKeyType{}, cfg)
}

// GetTableConfig retrieves the *TableConfig from context, or nil.
func GetTableConfig(ctx context.Context) *TableConfig {
	cfg, _ := ctx.Value(tableConfigKeyType{}).(*TableConfig)
	return cfg
}
