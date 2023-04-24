// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package alerts

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "alerts",
	Short: `The alerts API can be used to perform CRUD operations on alerts.`,
	Long: `The alerts API can be used to perform CRUD operations on alerts. An alert is a
  Databricks SQL object that periodically runs a query, evaluates a condition of
  its result, and notifies one or more users and/or notification destinations if
  the condition was met.`,
}

// start create command

var createReq sql.CreateAlert
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Parent, "parent", createReq.Parent, `The identifier of the workspace folder containing the alert.`)
	createCmd.Flags().IntVar(&createReq.Rearm, "rearm", createReq.Rearm, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create an alert.`,
	Long: `Create an alert.
  
  Creates an alert. An alert is a Databricks SQL object that periodically runs a
  query, evaluates a condition of its result, and notifies users or notification
  destinations if the condition was met.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.Name = args[0]
		_, err = fmt.Sscan(args[1], &createReq.Options)
		if err != nil {
			return fmt.Errorf("invalid OPTIONS: %s", args[1])
		}
		createReq.QueryId = args[2]

		response, err := w.Alerts.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq sql.DeleteAlertRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete ALERT_ID",
	Short: `Delete an alert.`,
	Long: `Delete an alert.
  
  Deletes an alert. Deleted alerts are no longer accessible and cannot be
  restored. **Note:** Unlike queries and dashboards, alerts cannot be moved to
  the trash.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Alerts.AlertNameToIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		deleteReq.AlertId = args[0]

		err = w.Alerts.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq sql.GetAlertRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get ALERT_ID",
	Short: `Get an alert.`,
	Long: `Get an alert.
  
  Gets an alert.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Alerts.AlertNameToIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		getReq.AlertId = args[0]

		response, err := w.Alerts.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get alerts.`,
	Long: `Get alerts.
  
  Gets a list of alerts.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Alerts.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq sql.EditAlert
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().IntVar(&updateReq.Rearm, "rearm", updateReq.Rearm, `Number of seconds after being triggered before the alert rearms itself and can be triggered again.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update an alert.`,
	Long: `Update an alert.
  
  Updates an alert.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.Name = args[0]
		_, err = fmt.Sscan(args[1], &updateReq.Options)
		if err != nil {
			return fmt.Errorf("invalid OPTIONS: %s", args[1])
		}
		updateReq.QueryId = args[2]
		updateReq.AlertId = args[3]

		err = w.Alerts.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Alerts
