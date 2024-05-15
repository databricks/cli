package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/spf13/cobra"
)

type tokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func buildLoginCommand(profile string, persistentAuth *auth.PersistentAuth) string {
	executable := os.Args[0]
	cmd := []string{
		executable,
		"auth",
		"login",
	}
	if profile != "" {
		cmd = append(cmd, "--profile", profile)
	} else {
		cmd = append(cmd, "--host", persistentAuth.Host)
		if persistentAuth.AccountID != "" {
			cmd = append(cmd, "--account-id", persistentAuth.AccountID)
		}
	}
	return strings.Join(cmd, " ")
}

func newTokenCommand(persistentAuth *auth.PersistentAuth) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token [HOST]",
		Short: "Get authentication token",
	}

	var tokenTimeout time.Duration
	cmd.Flags().DurationVar(&tokenTimeout, "timeout", auth.DefaultTimeout,
		"Timeout for acquiring a token.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		var profileName string
		profileFlag := cmd.Flag("profile")
		if profileFlag != nil {
			profileName = profileFlag.Value.String()
			// If a profile is provided we read the host from the .databrickscfg file
			if profileName != "" && len(args) > 0 {
				return errors.New("providing both a profile and a host parameters is not supported")
			}
		}

		err := setHostAndAccountId(ctx, profileName, persistentAuth, args)
		if err != nil {
			return err
		}
		defer persistentAuth.Close()

		ctx, cancel := context.WithTimeout(ctx, tokenTimeout)
		defer cancel()
		t, err := persistentAuth.Load(ctx)
		var httpErr *httpclient.HttpError
		if errors.As(err, &httpErr) {
			t := &tokenErrorResponse{}
			err = json.Unmarshal([]byte(httpErr.Message), t)
			if err != nil {
				return fmt.Errorf("error parsing token response: %w", err)
			}
			if t.ErrorDescription == "Refresh token is invalid" {
				return fmt.Errorf("a new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run `%s`", buildLoginCommand(profileName, persistentAuth))
			}
		} else if err != nil {
			return err
		}
		raw, err := json.MarshalIndent(t, "", "  ")
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(raw)
		return nil
	}

	return cmd
}
