package root

import (
	"log/slog"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/log/handler"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

// TelemetryAuthBudget bounds how long after its start a command waits for the
// background auth resolution its telemetry upload needs. A resolution still
// unfinished when the run ends and the budget has passed drops the events.
// Matches the upload's own default timeout. A run waits only once its command
// asks for it (TelemetryAuth.WaitAtExit).
var TelemetryAuthBudget = 3 * time.Second

// TelemetryAuthResolver resolves auth onto cmd's context. A seam for tests.
var TelemetryAuthResolver = func(cmd *cobra.Command, args []string) error {
	_, err := MustAnyClient(cmd, args)
	return err
}

// TelemetryAuth is the best-effort auth resolution of one command run.
type TelemetryAuth struct {
	pending *pendingAuth

	// WaitAtExit is whether the end of the run waits for an unfinished
	// resolution (within TelemetryAuthBudget). False by default: the
	// telemetry uploads only if the resolution has already finished, so a run
	// that ends early stays instant. RunE sets it once the run has gone far
	// enough to be worth the wait.
	WaitAtExit bool
}

// WithTelemetryAuth makes cmd resolve auth best-effort so its telemetry can
// upload at exit: PreRunE starts the resolution in the background, and RunE's
// wrapper uses it if it has finished, or waits for it within
// TelemetryAuthBudget once RunE sets WaitAtExit on the returned value. Nothing
// is resolved when telemetry is disabled. The command never fails or prompts
// because of it.
func WithTelemetryAuth(cmd *cobra.Command) *TelemetryAuth {
	auth := &TelemetryAuth{}
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		auth.pending = startTelemetryAuth(cmd, args)
		return nil
	}
	runE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := runE(cmd, args)
		auth.pending.apply(cmd, auth.WaitAtExit)
		return err
	}
	return auth
}

// pendingAuth is a background auth resolution started by startTelemetryAuth.
type pendingAuth struct {
	done     chan struct{}
	deadline time.Time
	cfg      *config.Config
}

// startTelemetryAuth resolves auth in the background on a shadow command. The
// shadow shares cmd's flags (for --profile) but not its context, so a
// resolution that outlives the budget never touches cmd's context. It logs at
// Debug only, so its discarded errors do not interleave with the command's
// output and prompts. It never refreshes an OAuth token (SkipTokenRefresh),
// as the command may exit before a refresh finishes; an expired cached token
// fails the resolution and drops the events.
func startTelemetryAuth(cmd *cobra.Command, args []string) *pendingAuth {
	ctx := cmd.Context()
	if telemetry.Disabled(ctx) {
		return nil
	}
	sctx := log.NewContext(ctx, slog.New(handler.DebugOnlyHandler{Handler: log.GetLogger(ctx).Handler()}))
	shadow := &cobra.Command{Use: cmd.Use}
	shadow.Flags().AddFlagSet(cmd.Flags())
	shadow.SetContext(SkipTokenRefresh(SkipLoadBundle(SkipPrompt(sctx))))

	p := &pendingAuth{done: make(chan struct{}), deadline: time.Now().Add(TelemetryAuthBudget)}
	resolve := TelemetryAuthResolver
	go func() {
		defer close(p.done)
		defer func() {
			if r := recover(); r != nil {
				log.Debugf(sctx, "auth for telemetry panicked: %v", r)
			}
		}()
		if err := resolve(shadow, args); err != nil {
			log.Debugf(sctx, "auth for telemetry failed: %v", err)
			return
		}
		if resolved := shadow.Context(); cmdctx.HasConfigUsed(resolved) {
			p.cfg = cmdctx.ConfigUsed(resolved)
		}
	}()
	return p
}

// apply puts the resolved config on cmd's context for the upload if the
// resolution has finished, waiting for it until the deadline when wait is set.
// A run that logged no event has nothing to upload, so it does not wait. Only
// the config is copied: the upload needs nothing else, and a workspace client
// on the context would change how root decorates the command's own errors.
func (p *pendingAuth) apply(cmd *cobra.Command, wait bool) {
	if p == nil {
		return
	}
	if len(telemetry.BufferedEvents(cmd.Context())) == 0 {
		return
	}
	if !wait && !p.finished() {
		log.Debugf(cmd.Context(), "auth for telemetry not resolved yet; not uploading telemetry")
		return
	}
	if wait && !p.wait() {
		log.Debugf(cmd.Context(), "auth for telemetry not resolved within %s; not uploading telemetry", TelemetryAuthBudget)
		return
	}
	if p.cfg != nil {
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), p.cfg))
	}
}

// wait reports whether the resolution finished by the deadline. A resolution
// that finished before the wait starts always counts: once a command outlasts
// the budget, a select on both channels would pick between them at random.
func (p *pendingAuth) wait() bool {
	if p.finished() {
		return true
	}
	timer := time.NewTimer(time.Until(p.deadline))
	defer timer.Stop()
	select {
	case <-p.done:
		return true
	case <-timer.C:
		return false
	}
}

// finished reports whether the resolution has finished, without waiting.
func (p *pendingAuth) finished() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}
