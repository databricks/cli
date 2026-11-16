package serving_endpoints

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/spf13/cobra"
)

// waitProgress reports progress while create or update-config waits for the
// endpoint to finish updating.
//
// The generated wait only shows state.config_update, which stays IN_PROGRESS
// for the whole deployment. Each served entity carries a more useful
// deployment_state_message (for example "Container image creation in progress"
// or "Provisioning"), so we show that instead, with the time waited so far.
//
// A spinner prints nothing when stderr is not a terminal, which left agents and
// CI with no output for the full deployment. In that case we print one line to
// stderr each time the message changes. Stdout is unchanged either way.
type waitProgress struct {
	start   time.Time
	spinner interface {
		Update(string)
		Close()
	}
	last string
}

func newWaitProgress(ctx context.Context) *waitProgress {
	p := &waitProgress{start: time.Now()}
	if cmdio.GetInteractiveMode(ctx) != cmdio.InteractiveModeNone {
		p.spinner = cmdio.NewSpinner(ctx)
	}
	return p
}

func (p *waitProgress) report(ctx context.Context, i *serving.ServingEndpointDetailed) {
	msg := endpointProgressMessage(i)
	line := fmt.Sprintf("%s (%s)", msg, time.Since(p.start).Truncate(time.Second))
	if p.spinner != nil {
		p.spinner.Update(line)
		return
	}
	// Without a spinner, print a line only when the message changes, not on every
	// poll. Skip the final poll: the update is done and the result follows on
	// stdout, so commands that finish immediately print exactly what they used to.
	if i.State != nil && i.State.ConfigUpdate == serving.EndpointStateConfigUpdateNotUpdating {
		return
	}
	if msg != p.last {
		p.last = msg
		cmdio.LogProgress(ctx, line)
	}
}

func (p *waitProgress) close() {
	if p.spinner != nil {
		p.spinner.Close()
	}
}

// endpointProgressMessage describes where the endpoint is in its update. It
// prefers the per-entity state of the pending config, falls back to the active
// config, and falls back to the generated "current status" text when no entity
// reports a state.
func endpointProgressMessage(i *serving.ServingEndpointDetailed) string {
	var parts []string
	if i != nil {
		var entities []serving.ServedEntityOutput
		if i.PendingConfig != nil {
			entities = i.PendingConfig.ServedEntities
		} else if i.Config != nil {
			entities = i.Config.ServedEntities
		}
		for _, e := range entities {
			if e.State == nil {
				continue
			}
			state := e.State.DeploymentStateMessage
			if state == "" {
				state = string(e.State.Deployment)
			}
			if state == "" {
				continue
			}
			if e.Name != "" {
				state = e.Name + ": " + state
			}
			parts = append(parts, state)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, "; ")
	}
	var status serving.EndpointStateConfigUpdate
	if i != nil && i.State != nil {
		status = i.State.ConfigUpdate
	}
	return fmt.Sprintf("current status: %s", status)
}

// unmarshalJSONFlag applies --json to req the same way the generated command
// does, including rendering non-fatal diagnostics such as unknown fields.
func unmarshalJSONFlag(cmd *cobra.Command, req any) error {
	if !cmd.Flags().Changed("json") {
		return nil
	}
	jsonFlag, ok := cmd.Flags().Lookup("json").Value.(*flags.JsonFlag)
	if !ok {
		return errors.New("unexpected type for --json flag")
	}
	diags := jsonFlag.Unmarshal(req)
	if diags.HasError() {
		return diags.Error()
	}
	if len(diags) > 0 {
		return cmdio.RenderDiagnostics(cmd.Context(), diags)
	}
	return nil
}

func waitFlags(cmd *cobra.Command) (skipWait bool, timeout time.Duration, err error) {
	skipWait, err = cmd.Flags().GetBool("no-wait")
	if err != nil {
		return false, 0, err
	}
	timeout, err = cmd.Flags().GetDuration("timeout")
	return skipWait, timeout, err
}

const waitHelp = `

  By default this command waits until the endpoint finishes updating (up to
  --timeout, 20m by default), printing each served entity's deployment state
  to stderr. Use --no-wait to return immediately, then check progress with
  "databricks serving-endpoints get NAME".`

func createOverride(createCmd *cobra.Command, createReq *serving.CreateServingEndpoint) {
	createCmd.Long += waitHelp
	createCmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if err := unmarshalJSONFlag(cmd, createReq); err != nil {
			return err
		}
		if !cmd.Flags().Changed("json") {
			createReq.Name = args[0]
		}
		skipWait, timeout, err := waitFlags(cmd)
		if err != nil {
			return err
		}

		wait, err := w.ServingEndpoints.Create(ctx, *createReq)
		if err != nil {
			return err
		}
		if skipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		p := newWaitProgress(ctx)
		info, err := wait.OnProgress(func(i *serving.ServingEndpointDetailed) {
			p.report(ctx, i)
		}).GetWithTimeout(timeout)
		p.close()
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	}
}

func updateConfigOverride(updateConfigCmd *cobra.Command, updateConfigReq *serving.EndpointCoreConfigInput) {
	updateConfigCmd.Long += waitHelp
	updateConfigCmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if err := unmarshalJSONFlag(cmd, updateConfigReq); err != nil {
			return err
		}
		updateConfigReq.Name = args[0]
		skipWait, timeout, err := waitFlags(cmd)
		if err != nil {
			return err
		}

		wait, err := w.ServingEndpoints.UpdateConfig(ctx, *updateConfigReq)
		if err != nil {
			return err
		}
		if skipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		p := newWaitProgress(ctx)
		info, err := wait.OnProgress(func(i *serving.ServingEndpointDetailed) {
			p.report(ctx, i)
		}).GetWithTimeout(timeout)
		p.close()
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	}
}

func init() {
	createOverrides = append(createOverrides, createOverride)
	updateConfigOverrides = append(updateConfigOverrides, updateConfigOverride)
}
