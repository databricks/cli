// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package domains

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/domains"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: `*Beta* Manage domains for organizing and discovering data assets.`,
		Long: `This command is in Beta and may change without notice.

Manage domains for organizing and discovering data assets.`,
		GroupID: "domains",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	// Add methods
	cmd.AddCommand(newCreateDomain())
	cmd.AddCommand(newDeleteDomain())
	cmd.AddCommand(newGetDomain())
	cmd.AddCommand(newListDomains())
	cmd.AddCommand(newUpdateDomain())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-domain command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDomainOverrides []func(
	*cobra.Command,
	*domains.CreateDomainRequest,
)

func newCreateDomain() *cobra.Command {
	cmd := &cobra.Command{}

	var createDomainReq domains.CreateDomainRequest
	createDomainReq.Domain = domains.Domain{}
	var createDomainJson flags.JsonFlag

	cmd.Flags().Var(&createDomainJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDomainReq.DomainId, "domain-id", createDomainReq.DomainId, `Client-supplied resource ID for the new domain.`)
	// TODO: array: business_owner_ids
	cmd.Flags().StringVar(&createDomainReq.Domain.Description, "description", createDomainReq.Domain.Description, `Full description (max 4096 chars).`)
	cmd.Flags().BoolVar(&createDomainReq.Domain.Draft, "draft", createDomainReq.Domain.Draft, `Whether to mark the domain as a draft.`)
	// TODO: complex arg: icon
	cmd.Flags().StringVar(&createDomainReq.Domain.Name, "name", createDomainReq.Domain.Name, `Full resource name of the domain.`)
	cmd.Flags().StringVar(&createDomainReq.Domain.ParentDomainId, "parent-domain-id", createDomainReq.Domain.ParentDomainId, `Domain ID of the parent.`)
	cmd.Flags().StringVar(&createDomainReq.Domain.Subtitle, "subtitle", createDomainReq.Domain.Subtitle, `Short description (max 280 chars).`)
	// TODO: array: technical_owner_ids

	cmd.Use = "create-domain TAG_KEY"
	cmd.Short = `*Beta* Create a domain.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a domain.

  Create a domain. If domain_id is omitted, the server generates one.

  Arguments:
    TAG_KEY: Governed tag key associated with this domain.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'tag_key' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createDomainJson.Unmarshal(&createDomainReq.Domain)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createDomainReq.Domain.TagKey = args[0]
		}

		response, err := w.Domains.CreateDomain(ctx, createDomainReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDomainOverrides {
		fn(cmd, &createDomainReq)
	}

	return cmd
}

// start delete-domain command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDomainOverrides []func(
	*cobra.Command,
	*domains.DeleteDomainRequest,
)

func newDeleteDomain() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDomainReq domains.DeleteDomainRequest

	cmd.Flags().BoolVar(&deleteDomainReq.Force, "force", deleteDomainReq.Force, `When false (default), DeleteDomain is rejected with FAILED_PRECONDITION if the domain still has Glossary pages.`)

	cmd.Use = "delete-domain NAME"
	cmd.Short = `*Beta* Delete a domain.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a domain.

  Delete a domain. By default the request fails if the domain still has Glossary
  pages; set force to delete those pages along with the domain.

  Arguments:
    NAME: Full resource name of the domain to delete. Format: domains/{domain_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDomainReq.Name = args[0]

		err = w.Domains.DeleteDomain(ctx, deleteDomainReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDomainOverrides {
		fn(cmd, &deleteDomainReq)
	}

	return cmd
}

// start get-domain command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDomainOverrides []func(
	*cobra.Command,
	*domains.GetDomainRequest,
)

func newGetDomain() *cobra.Command {
	cmd := &cobra.Command{}

	var getDomainReq domains.GetDomainRequest

	cmd.Use = "get-domain NAME"
	cmd.Short = `*Beta* Get a domain.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a domain.

  Get a domain by resource name.

  Authorization: external callers must have the MANAGE DISCOVERY permission.

  Arguments:
    NAME: Full resource name of the domain to retrieve. Format:
      domains/{domain_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDomainReq.Name = args[0]

		response, err := w.Domains.GetDomain(ctx, getDomainReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDomainOverrides {
		fn(cmd, &getDomainReq)
	}

	return cmd
}

// start list-domains command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDomainsOverrides []func(
	*cobra.Command,
	*domains.ListDomainsRequest,
)

func newListDomains() *cobra.Command {
	cmd := &cobra.Command{}

	var listDomainsReq domains.ListDomainsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDomainsLimit int

	cmd.Flags().IntVar(&listDomainsReq.PageSize, "page-size", listDomainsReq.PageSize, ``)
	cmd.Flags().StringVar(&listDomainsReq.ParentDomainId, "parent-domain-id", listDomainsReq.ParentDomainId, `Filter by parent domain.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDomainsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listDomainsReq.PageToken, "page-token", listDomainsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-domains"
	cmd.Short = `*Beta* List domains.`
	cmd.Long = `This command is in Beta and may change without notice.

List domains.

  List domains in the account. Set parent_domain_id to return only the direct
  subdomains of a given domain.

  Authorization: external callers must have the MANAGE DISCOVERY permission;
  only domains the caller is authorized to read are returned.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Domains.ListDomains(ctx, listDomainsReq)
		if listDomainsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDomainsLimit)
		}
		if listDomainsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDomainsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDomainsOverrides {
		fn(cmd, &listDomainsReq)
	}

	return cmd
}

// start update-domain command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDomainOverrides []func(
	*cobra.Command,
	*domains.UpdateDomainRequest,
)

func newUpdateDomain() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDomainReq domains.UpdateDomainRequest
	updateDomainReq.Domain = domains.Domain{}
	var updateDomainJson flags.JsonFlag

	cmd.Flags().Var(&updateDomainJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: business_owner_ids
	cmd.Flags().StringVar(&updateDomainReq.Domain.Description, "description", updateDomainReq.Domain.Description, `Full description (max 4096 chars).`)
	cmd.Flags().BoolVar(&updateDomainReq.Domain.Draft, "draft", updateDomainReq.Domain.Draft, `Whether to mark the domain as a draft.`)
	// TODO: complex arg: icon
	cmd.Flags().StringVar(&updateDomainReq.Domain.Name, "name", updateDomainReq.Domain.Name, `Full resource name of the domain.`)
	cmd.Flags().StringVar(&updateDomainReq.Domain.ParentDomainId, "parent-domain-id", updateDomainReq.Domain.ParentDomainId, `Domain ID of the parent.`)
	cmd.Flags().StringVar(&updateDomainReq.Domain.Subtitle, "subtitle", updateDomainReq.Domain.Subtitle, `Short description (max 280 chars).`)
	// TODO: array: technical_owner_ids

	cmd.Use = "update-domain NAME UPDATE_MASK TAG_KEY"
	cmd.Short = `*Beta* Update a domain.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a domain.

  Update a domain. update_mask selects which fields to modify; the domain is
  identified by its resource name.

  Arguments:
    NAME: Full resource name of the domain. The primary identifier for this
      resource. Format: domains/{domain_id} Identifies the domain on get,
      update, and delete. Not an input on create — to choose the id, set
      CreateDomainRequest.domain_id.
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.
    TAG_KEY: Governed tag key associated with this domain.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'tag_key' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateDomainJson.Unmarshal(&updateDomainReq.Domain)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateDomainReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateDomainReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updateDomainReq.Domain.TagKey = args[2]
		}

		response, err := w.Domains.UpdateDomain(ctx, updateDomainReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDomainOverrides {
		fn(cmd, &updateDomainReq)
	}

	return cmd
}

// end service Domains
