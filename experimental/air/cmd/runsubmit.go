package aircmd

// This file used to hold the ephemeral runs/submit path (submitWorkload +
// buildSubmitPayload). That path was retired: `air run` now submits through DABs
// (convert -> deploy -> run, see rundabs.go), per the AIR-CLI/DABs agreement. What
// remains here is the runtime-image defaults shared by the DABs converter
// (exportbundle.go) — the one piece of the old submit path the bundle path needs.

// defaultDlRuntimeImage is the default deep-learning runtime channel. The converter
// strips the CLIENT-GPU- prefix; unlike the retired submit path it does not read a
// process-env override, so a generated bundle is reproducible.
const defaultDlRuntimeImage = "CLIENT-GPU-4"

// aiRuntimeEnvironmentKey ties the task to the serverless environment that carries
// the runtime channel.
const aiRuntimeEnvironmentKey = "default"
