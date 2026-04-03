package agent

import (
	"github.com/spf13/cobra"
)

// CheckConsentForFlags validates agent consent for gated flags that have been
// explicitly set on the command. Returns nil if no agent is detected or if
// no gated flags are set.
func CheckConsentForFlags(cmd *cobra.Command) error {
	ctx := cmd.Context()

	// Non-agent callers are not gated.
	if Product(ctx) == "" {
		return nil
	}

	// Map of flag names to the consent operation they require.
	flagOps := map[string]string{
		"force-lock":   OperationForceLock,
		"auto-approve": OperationAutoApprove,
		"force":        OperationForceDeploy,
	}

	for flagName, operation := range flagOps {
		f := cmd.Flag(flagName)
		if f == nil || !f.Changed {
			continue
		}
		if err := ValidateConsent(ctx, operation); err != nil {
			return err
		}
	}

	return nil
}
