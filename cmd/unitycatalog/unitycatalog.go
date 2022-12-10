// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package unitycatalog

import (
	"github.com/databricks/bricks/cmd/unitycatalog/catalogs"
	external_locations "github.com/databricks/bricks/cmd/unitycatalog/external-locations"
	"github.com/databricks/bricks/cmd/unitycatalog/grants"
	"github.com/databricks/bricks/cmd/unitycatalog/metastores"
	"github.com/databricks/bricks/cmd/unitycatalog/providers"
	recipient_activation "github.com/databricks/bricks/cmd/unitycatalog/recipient-activation"
	"github.com/databricks/bricks/cmd/unitycatalog/recipients"
	"github.com/databricks/bricks/cmd/unitycatalog/schemas"
	"github.com/databricks/bricks/cmd/unitycatalog/shares"
	storage_credentials "github.com/databricks/bricks/cmd/unitycatalog/storage-credentials"
	"github.com/databricks/bricks/cmd/unitycatalog/tables"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "unitycatalog",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(catalogs.Cmd)
	Cmd.AddCommand(external_locations.Cmd)
	Cmd.AddCommand(grants.Cmd)
	Cmd.AddCommand(metastores.Cmd)
	Cmd.AddCommand(providers.Cmd)
	Cmd.AddCommand(recipient_activation.Cmd)
	Cmd.AddCommand(recipients.Cmd)
	Cmd.AddCommand(schemas.Cmd)
	Cmd.AddCommand(shares.Cmd)
	Cmd.AddCommand(storage_credentials.Cmd)
	Cmd.AddCommand(tables.Cmd)
}
