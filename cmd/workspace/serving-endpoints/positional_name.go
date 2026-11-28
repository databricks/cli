package serving_endpoints

// This file's overrides wrap RunE, so it must sort after any file whose
// overrides replace RunE outright: overrides run in init() order, which follows
// file names.

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/spf13/cobra"
)

const nameWithJSONHelp = `

  With --json, the endpoint NAME can be passed either as the argument or as
  "name" in the JSON.`

// jsonName returns the top-level "name" of --json, or "" if --json isn't set,
// has no name, or isn't a JSON object (the generated code reports bad JSON).
func jsonName(cmd *cobra.Command) string {
	jsonFlag, ok := cmd.Flags().Lookup("json").Value.(*flags.JsonFlag)
	if !ok || !cmd.Flags().Changed("json") {
		return ""
	}
	var body struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(jsonFlag.Raw(), &body)
	return body.Name
}

// atMostOneArgWithJSON allows NAME to be omitted when --json is set, and
// requires it otherwise.
func atMostOneArgWithJSON(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("json") && len(args) == 0 {
		return nil
	}
	return root.ExactArgs(1)(cmd, args)
}

// createNameOverride lets create take NAME as an argument together with
// --json. The generated command rejected that ("no positional arguments are
// allowed"), even though the usage line is "create NAME" and update-config
// requires the argument, so users and agents got one of the two wrong.
func createNameOverride(createCmd *cobra.Command, createReq *serving.CreateServingEndpoint) {
	createCmd.Long += nameWithJSONHelp
	createCmd.Args = atMostOneArgWithJSON
	generatedRunE := createCmd.RunE
	createCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") && len(args) == 1 {
			if name := jsonName(cmd); name != "" && name != args[0] {
				return fmt.Errorf("the endpoint name %q doesn't match \"name\" %q in --json; pass it in one place only", args[0], name)
			}
			// --json is decoded into createReq after this, and keeps fields
			// it doesn't set.
			createReq.Name = args[0]
		}
		return generatedRunE(cmd, args)
	}
}

// updateConfigNameOverride lets update-config take the endpoint name from
// "name" in --json when NAME isn't passed, matching what create accepts.
func updateConfigNameOverride(updateConfigCmd *cobra.Command, _ *serving.EndpointCoreConfigInput) {
	updateConfigCmd.Long += nameWithJSONHelp
	updateConfigCmd.Args = atMostOneArgWithJSON
	generatedRunE := updateConfigCmd.RunE
	updateConfigCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			name, err := takeJSONName(cmd)
			if err != nil {
				return err
			}
			args = []string{name}
		}
		return generatedRunE(cmd, args)
	}
}

// takeJSONName returns "name" from --json and removes it from the body. The
// update-config body has no "name" field, so leaving it in would print an
// "unknown field" warning for a value that was used.
func takeJSONName(cmd *cobra.Command) (string, error) {
	jsonFlag, ok := cmd.Flags().Lookup("json").Value.(*flags.JsonFlag)
	if !ok {
		return "", errors.New("unexpected type for --json flag")
	}
	// Without a usable name, fail exactly as the generated command did when
	// NAME was missing.
	missingName := root.ExactArgs(1)(cmd, nil)
	var body map[string]json.RawMessage
	if err := json.Unmarshal(jsonFlag.Raw(), &body); err != nil {
		return "", missingName
	}
	var name string
	if err := json.Unmarshal(body["name"], &name); err != nil || name == "" {
		return "", missingName
	}
	delete(body, "name")
	stripped, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return name, jsonFlag.Set(string(stripped))
}

func init() {
	createOverrides = append(createOverrides, createNameOverride)
	updateConfigOverrides = append(updateConfigOverrides, updateConfigNameOverride)
}
