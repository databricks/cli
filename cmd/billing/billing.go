package billing

import (
	billable_usage "github.com/databricks/bricks/cmd/billing/billable-usage"
	"github.com/databricks/bricks/cmd/billing/budgets"
	log_delivery "github.com/databricks/bricks/cmd/billing/log-delivery"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "billing",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(billable_usage.Cmd)
	Cmd.AddCommand(budgets.Cmd)
	Cmd.AddCommand(log_delivery.Cmd)
}
