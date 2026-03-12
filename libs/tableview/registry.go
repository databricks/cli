package tableview

import (
	"sync"

	"github.com/spf13/cobra"
)

var (
	configMu sync.RWMutex
	configs  = map[*cobra.Command]*TableConfig{}
)

// RegisterConfig associates a TableConfig with a command.
func RegisterConfig(cmd *cobra.Command, cfg TableConfig) {
	configMu.Lock()
	defer configMu.Unlock()
	configs[cmd] = &cfg
}

// GetConfig retrieves the TableConfig for a command, if registered.
func GetConfig(cmd *cobra.Command) *TableConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return configs[cmd]
}
