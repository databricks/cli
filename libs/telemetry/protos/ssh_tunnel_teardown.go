package protos

// SshTunnelTeardownEvent is emitted by the SSH tunnel server on the compute when it
// shuts down after its idle timeout. It is a separate event from SshTunnelEvent
// because it is not a connection attempt: folding it into that event would add rows
// that every existing is_success query would count as connections.
//
// It exists to size the problem the --keep-detached-for flag addresses: only the
// server, running on the compute at teardown, can see whether the session left
// detached processes behind, and by then the client that started it is long gone.
//
// The linger itself is deliberately not reported here. The bootstrap notebook is what
// holds the run open, and it outlives every Go process in the session, so no CLI
// process can observe when the linger ends; the run's own duration carries that.
type SshTunnelTeardownEvent struct {
	// Type of compute: dedicated cluster or serverless.
	ComputeType SshTunnelComputeType `json:"compute_type,omitempty"`

	// Whether the session asked for detached processes to be kept via
	// --keep-detached-for. Only the presence is recorded, not the duration.
	KeepDetachedRequested bool `json:"keep_detached_requested"`

	// Whether processes the tunnel started, but that left its process group (tmux,
	// setsid, nohup), were still running when the server shut down. Without
	// --keep-detached-for those processes do not survive the run, so this counts how
	// often the tunnel destroys work a user meant to keep.
	HadDetachedDescendantsAtTeardown bool `json:"had_detached_descendants_at_teardown"`
}
