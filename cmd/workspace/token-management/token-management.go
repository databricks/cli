// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package token_management

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "token-management",
	Short: `Enables administrators to get all tokens and delete tokens for other users.`,
	Long: `Enables administrators to get all tokens and delete tokens for other users.
  Admins can either get every token, get a specific token by ID, or get all
  tokens for a particular user.`,
}

// start create-obo-token command

var createOboTokenReq settings.CreateOboTokenRequest
var createOboTokenJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createOboTokenCmd)
	// TODO: short flags
	createOboTokenCmd.Flags().Var(&createOboTokenJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createOboTokenCmd.Flags().StringVar(&createOboTokenReq.Comment, "comment", createOboTokenReq.Comment, `Comment that describes the purpose of the token.`)

}

var createOboTokenCmd = &cobra.Command{
	Use:   "create-obo-token APPLICATION_ID LIFETIME_SECONDS",
	Short: `Create on-behalf token.`,
	Long: `Create on-behalf token.
  
  Creates a token on behalf of a service principal.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createOboTokenJson.Unmarshal(&createOboTokenReq)
			if err != nil {
				return err
			}
		} else {
			createOboTokenReq.ApplicationId = args[0]
			_, err = fmt.Sscan(args[1], &createOboTokenReq.LifetimeSeconds)
			if err != nil {
				return fmt.Errorf("invalid LIFETIME_SECONDS: %s", args[1])
			}
		}

		response, err := w.TokenManagement.CreateOboToken(ctx, createOboTokenReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq settings.DeleteTokenManagementRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete TOKEN_ID",
	Short: `Delete a token.`,
	Long: `Delete a token.
  
  Deletes a token, specified by its ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No TOKEN_ID argument specified. Loading names for Token Management drop-down."
				names, err := w.TokenManagement.TokenInfoCommentToTokenIdMap(ctx, settings.ListTokenManagementRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Token Management drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID of the token to get")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the id of the token to get")
			}
			deleteReq.TokenId = args[0]
		}

		err = w.TokenManagement.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq settings.GetTokenManagementRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get TOKEN_ID",
	Short: `Get token info.`,
	Long: `Get token info.
  
  Gets information about a token, specified by its ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No TOKEN_ID argument specified. Loading names for Token Management drop-down."
				names, err := w.TokenManagement.TokenInfoCommentToTokenIdMap(ctx, settings.ListTokenManagementRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Token Management drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID of the token to get")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the id of the token to get")
			}
			getReq.TokenId = args[0]
		}

		response, err := w.TokenManagement.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq settings.ListTokenManagementRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listCmd.Flags().StringVar(&listReq.CreatedById, "created-by-id", listReq.CreatedById, `User ID of the user that created the token.`)
	listCmd.Flags().StringVar(&listReq.CreatedByUsername, "created-by-username", listReq.CreatedByUsername, `Username of the user that created the token.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List all tokens.`,
	Long: `List all tokens.
  
  Lists all tokens associated with the specified workspace or user.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.TokenManagement.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service TokenManagement
