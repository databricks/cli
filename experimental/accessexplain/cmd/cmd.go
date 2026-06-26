// Package cmd wires the access-explain engine into the experimental command
// tree. It reads effective permissions and masking policies via the SDK and
// feeds a plain trace to accessexplain.Evaluate, which computes the verdict.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/accessexplain"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// accessReader supplies the effective permissions, column masks, and current
// user for the trace. It is an interface so the command orchestration can be
// tested without a live workspace.
type accessReader interface {
	Effective(ctx context.Context, securableType, fullName, principal string) ([]accessexplain.HeldPrivilege, error)
	ColumnMasks(ctx context.Context, tableFullName, principal string) ([]accessexplain.Mask, error)
	LegacyColumnMasks(ctx context.Context, tableFullName string) ([]accessexplain.Mask, error)
	CurrentUser(ctx context.Context) (string, error)
	// ResolvePrincipal reports a principal's type ("user", "group", or
	// "service principal") and whether it exists. A non-nil error means
	// existence could not be determined (e.g. the caller lacks list permission),
	// which callers treat as "proceed without verifying".
	ResolvePrincipal(ctx context.Context, name string) (kind string, found bool, err error)
}

// New returns the experimental "access" command group with the explain subcommand.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access",
		Short: "Access (authorization) diagnostics",
		Long:  "Access diagnostics that trace why a principal can or cannot access a Unity Catalog securable.",
	}
	cmd.AddCommand(newExplainCommand())
	return cmd
}

func newExplainCommand() *cobra.Command {
	var (
		on        string
		principal string
		privilege string
	)

	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Explain why a principal can or cannot access a Unity Catalog securable",
		Long: `Trace the access decision for a Unity Catalog securable and explain the
verdict, with the exact GRANT to fix a denial.

The securable is a dotted name: a catalog, catalog.schema, or
catalog.schema.table. Access requires USE CATALOG, USE SCHEMA, and the leaf
privilege (SELECT for a table by default). Without --principal the current user
is used; --principal requires the caller to be privileged on the securable
(Unity Catalog has no impersonation).`,
		Args:    cobra.NoArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			if on == "" {
				return errors.New("--on is required (catalog[.schema[.table]])")
			}
			ctx := cmd.Context()
			reader := &sdkReader{w: cmdctx.WorkspaceClient(ctx)}
			verdict, err := explain(ctx, reader, on, principal, privilege)
			if err != nil {
				return err
			}
			return renderVerdict(ctx, cmd, verdict)
		},
	}

	cmd.Flags().StringVar(&on, "on", "", "Securable to explain: catalog[.schema[.table]]")
	cmd.Flags().StringVar(&principal, "principal", "", "Principal to explain access for (default: current user)")
	cmd.Flags().StringVar(&privilege, "privilege", "", "Privilege to check at the leaf securable (default: SELECT for tables)")
	return cmd
}

// explain assembles the trace and computes the verdict.
func explain(ctx context.Context, reader accessReader, on, principal, privilege string) (accessexplain.Verdict, error) {
	specs, err := accessexplain.ParseSecurable(on, privilege)
	if err != nil {
		return accessexplain.Verdict{}, err
	}

	var principalKind string
	if principal == "" {
		principal, err = reader.CurrentUser(ctx)
		if err != nil {
			return accessexplain.Verdict{}, fmt.Errorf("resolve current user: %w", err)
		}
	} else {
		// An explicitly named principal is verified so a typo reads as "not
		// found" instead of a misleading DENIED with no privileges anywhere.
		kind, found, rerr := reader.ResolvePrincipal(ctx, principal)
		switch {
		case rerr != nil:
			// Could not verify (e.g. no list permission); proceed unverified.
			log.Debugf(ctx, "access explain: could not verify principal %q: %v", principal, rerr)
		case !found:
			return accessexplain.Verdict{}, fmt.Errorf("principal %q was not found as a user, group, or service principal", principal)
		default:
			principalKind = kind
		}
	}

	in := accessexplain.Input{
		Principal: principal,
		Securable: on,
		Action:    specs[len(specs)-1].Needed,
	}
	for _, spec := range specs {
		held, err := reader.Effective(ctx, spec.Type, spec.FullName, principal)
		if err != nil {
			return accessexplain.Verdict{}, fmt.Errorf("read effective permissions on %s %s: %w", spec.Type, spec.FullName, err)
		}
		in.Levels = append(in.Levels, accessexplain.Level{LevelSpec: spec, Held: held})
	}

	// Column masks only apply to a table leaf. They are supplementary to the
	// allow/deny verdict, so a failure to read them (commonly a missing READ
	// METADATA grant on the table) is logged and skipped, not fatal.
	if leaf := specs[len(specs)-1]; leaf.Type == accessexplain.SecurableTable {
		policyMasks, err := reader.ColumnMasks(ctx, leaf.FullName, principal)
		if err != nil {
			log.Warnf(ctx, "access explain: could not list masking policies on %s, mask detection skipped: %v", leaf.FullName, err)
		}
		legacyMasks, err := reader.LegacyColumnMasks(ctx, leaf.FullName)
		if err != nil {
			log.Warnf(ctx, "access explain: could not read column masks on %s, mask detection skipped: %v", leaf.FullName, err)
		}
		in.Masks = mergeMasks(policyMasks, legacyMasks)
	}

	verdict := accessexplain.Evaluate(in)
	verdict.PrincipalKind = principalKind
	return verdict, nil
}

// mergeMasks combines ABAC policy masks and legacy column masks, keeping at
// most one entry per column (the ABAC policy wins, since it is the active
// mechanism when both are present).
func mergeMasks(policyMasks, legacyMasks []accessexplain.Mask) []accessexplain.Mask {
	seen := map[string]bool{}
	var out []accessexplain.Mask
	for _, m := range append(policyMasks, legacyMasks...) {
		if seen[m.Column] {
			continue
		}
		seen[m.Column] = true
		out = append(out, m)
	}
	return out
}

// sdkReader is the production accessReader backed by a workspace client.
type sdkReader struct {
	w *databricks.WorkspaceClient
}

func (r *sdkReader) Effective(ctx context.Context, securableType, fullName, principal string) ([]accessexplain.HeldPrivilege, error) {
	var held []accessexplain.HeldPrivilege
	pageToken := ""
	for {
		resp, err := r.w.Grants.GetEffective(ctx, catalog.GetEffectiveRequest{
			SecurableType: securableType,
			FullName:      fullName,
			Principal:     principal,
			PageToken:     pageToken,
		})
		if err != nil {
			return nil, err
		}
		for _, pa := range resp.PrivilegeAssignments {
			for _, p := range pa.Privileges {
				held = append(held, accessexplain.HeldPrivilege{
					Name:              string(p.Privilege),
					InheritedFromType: string(p.InheritedFromType),
					InheritedFromName: p.InheritedFromName,
				})
			}
		}
		// Pagination is across principals; a missing or unchanged token ends it.
		if resp.NextPageToken == "" || resp.NextPageToken == pageToken {
			return held, nil
		}
		pageToken = resp.NextPageToken
	}
}

func (r *sdkReader) ColumnMasks(ctx context.Context, tableFullName, principal string) ([]accessexplain.Mask, error) {
	policies, err := listing.ToSlice(ctx, r.w.Policies.ListPolicies(ctx, catalog.ListPoliciesRequest{
		OnSecurableType:     accessexplain.SecurableTable,
		OnSecurableFullname: tableFullName,
		IncludeInherited:    true,
	}))
	if err != nil {
		return nil, err
	}

	var masks []accessexplain.Mask
	for _, p := range policies {
		if m, ok := columnMaskFromPolicy(p, principal); ok {
			masks = append(masks, m)
		}
	}
	return masks, nil
}

// columnMaskFromPolicy converts an ABAC policy to a Mask, reporting whether it
// should be surfaced. Only column-mask policies that do not explicitly except
// the principal are surfaced. A policy targeting a group is kept (with its
// Targets) rather than dropped, because group membership cannot be resolved
// locally and silently hiding a mask would understate what is masked.
func columnMaskFromPolicy(p catalog.PolicyInfo, principal string) (accessexplain.Mask, bool) {
	if p.PolicyType != catalog.PolicyTypePolicyTypeColumnMask || p.ColumnMask == nil {
		return accessexplain.Mask{}, false
	}
	if slices.Contains(p.ExceptPrincipals, principal) {
		return accessexplain.Mask{}, false
	}
	// Definitely applies when it targets everyone or names the principal
	// directly; otherwise it may only apply via an unresolved group target.
	applies := len(p.ToPrincipals) == 0 || slices.Contains(p.ToPrincipals, principal)
	return accessexplain.Mask{
		Column:   p.ColumnMask.OnColumn,
		Policy:   p.Name,
		Function: p.ColumnMask.FunctionName,
		Targets:  p.ToPrincipals,
		Applies:  applies,
	}, true
}

// LegacyColumnMasks reports the function-based column masks attached directly
// to the table's columns (ColumnInfo.Mask), the pre-ABAC masking mechanism.
// These have no policy name and apply to all readers, so no Targets are set.
// A read error is returned for the caller to treat as best-effort.
func (r *sdkReader) LegacyColumnMasks(ctx context.Context, tableFullName string) ([]accessexplain.Mask, error) {
	table, err := r.w.Tables.GetByFullName(ctx, tableFullName)
	if err != nil {
		return nil, err
	}
	var masks []accessexplain.Mask
	for _, c := range table.Columns {
		if c.Mask == nil {
			continue
		}
		masks = append(masks, accessexplain.Mask{
			Column:   c.Name,
			Function: c.Mask.FunctionName,
			// Legacy column masks are attached to the column itself and apply to
			// every reader.
			Applies: true,
		})
	}
	return masks, nil
}

func (r *sdkReader) CurrentUser(ctx context.Context) (string, error) {
	me, err := r.w.CurrentUser.Me(ctx, iam.MeRequest{})
	if err != nil {
		return "", err
	}
	return me.UserName, nil
}

// ResolvePrincipal looks the name up in SCIM as a user (by userName), group, or
// service principal (both by displayName). It only needs existence, so it reads
// the first match rather than draining every page. A list error on any lookup
// is returned so the caller can proceed unverified rather than reporting a real
// principal as missing.
func (r *sdkReader) ResolvePrincipal(ctx context.Context, name string) (string, bool, error) {
	userFilter := fmt.Sprintf("userName eq %q", name)
	if found, err := iteratorHasMatch(ctx, r.w.UsersV2.List(ctx, iam.ListUsersRequest{Filter: userFilter})); err != nil {
		return "", false, err
	} else if found {
		return "user", true, nil
	}

	nameFilter := fmt.Sprintf("displayName eq %q", name)
	if found, err := iteratorHasMatch(ctx, r.w.GroupsV2.List(ctx, iam.ListGroupsRequest{Filter: nameFilter})); err != nil {
		return "", false, err
	} else if found {
		return "group", true, nil
	}

	if found, err := iteratorHasMatch(ctx, r.w.ServicePrincipalsV2.List(ctx, iam.ListServicePrincipalsRequest{Filter: nameFilter})); err != nil {
		return "", false, err
	} else if found {
		return "service principal", true, nil
	}

	return "", false, nil
}

// iteratorHasMatch reports whether a listing iterator yields at least one item.
// It reads only the first item (no full drain) and surfaces a page-fetch error.
func iteratorHasMatch[T any](ctx context.Context, it listing.Iterator[T]) (bool, error) {
	if !it.HasNext(ctx) {
		return false, nil
	}
	// HasNext is also true when the page fetch failed; Next then returns the
	// error, which we propagate so the caller treats it as "could not verify".
	if _, err := it.Next(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func renderVerdict(ctx context.Context, cmd *cobra.Command, v accessexplain.Verdict) error {
	switch root.OutputType(cmd) {
	case flags.OutputText:
		renderText(ctx, cmd, v)
		return nil
	case flags.OutputJSON:
		buf, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(append(buf, '\n'))
		return err
	default:
		return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
	}
}

func renderText(ctx context.Context, cmd *cobra.Command, v accessexplain.Verdict) {
	out := cmd.OutOrStdout()

	decision := cmdio.Green(ctx, "ALLOWED")
	if !v.Allowed {
		decision = cmdio.Red(ctx, "DENIED")
	}
	principal := v.Principal
	if v.PrincipalKind != "" {
		principal += " (" + v.PrincipalKind + ")"
	}
	fmt.Fprintf(out, "%s for %s on %s (%s)\n", decision, cmdio.Bold(ctx, principal), cmdio.Bold(ctx, v.Securable), v.Action)

	fmt.Fprintln(out, "Why:")
	for _, l := range v.Levels {
		if l.Satisfied {
			fmt.Fprintf(out, "  %s %s %s: %s\n", cmdio.Green(ctx, "✓"), l.Type, l.FullName, l.SatisfiedBy)
		} else {
			fmt.Fprintf(out, "  %s %s %s: missing %s\n", cmdio.Red(ctx, "✗"), l.Type, l.FullName, l.Needed)
		}
	}

	if len(v.Masks) > 0 {
		fmt.Fprintln(out, "Masking:")
		for _, m := range v.Masks {
			verb := "is masked by"
			if !m.Applies {
				verb = "may be masked by"
			}
			fmt.Fprintf(out, "  %s column %s %s %s\n", cmdio.Yellow(ctx, "⚠"), m.Column, verb, maskSource(m))
		}
	}

	if len(v.Fixes) > 0 {
		fmt.Fprintln(out, "Fix:")
		for _, f := range v.Fixes {
			fmt.Fprintf(out, "  %s\n", cmdio.Cyan(ctx, f))
		}
	}
}

// maskSource describes what masks a column: an ABAC policy by name, or a legacy
// function-based mask (which has no policy name). A policy that targets specific
// principals/groups (not everyone) is annotated, since group membership is not
// resolved locally and the mask may or may not apply to this principal.
func maskSource(m accessexplain.Mask) string {
	var src string
	switch {
	case m.Policy != "":
		src = "policy " + m.Policy
	case m.Function != "":
		src = "function " + m.Function
	default:
		src = "a column mask"
	}
	if len(m.Targets) > 0 {
		src += fmt.Sprintf(" (targets %s)", strings.Join(m.Targets, ", "))
	}
	return src
}
