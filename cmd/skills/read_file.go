package skills

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
)

// readFileOutput is the JSON output of read-file. ContentBase64 holds the bytes
// of the file, base64-encoded.
type readFileOutput struct {
	Path          string `json:"path"`
	MediaType     string `json:"media_type"`
	ContentBase64 string `json:"content_base64"`
}

func newReadFileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read-file <name> <path>",
		Short: "Read one file of a skill",
		Args:  cobra.ExactArgs(2),
		RunE:  runReadFile,
	}
}

func runReadFile(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	src, err := libskills.Resolve(ctx)
	if err != nil {
		return err
	}
	name, file := args[0], args[1]
	data, err := src.ReadFile(ctx, name, file)
	if err != nil {
		return err
	}
	mediaType := http.DetectContentType(data)
	if root.OutputType(cmd) == flags.OutputJSON {
		return cmdio.Render(ctx, readFileOutput{
			Path:          file,
			MediaType:     mediaType,
			ContentBase64: base64.StdEncoding.EncodeToString(data),
		})
	}
	if !isText(data) {
		return fmt.Errorf("%q in skill %q is not text (%s); use --output json to read it base64-encoded with its media type", file, name, mediaType)
	}
	_, err = cmd.OutOrStdout().Write(data)
	return err
}

// isText reports whether data is text: valid UTF-8 with no NUL byte.
func isText(data []byte) bool {
	return utf8.Valid(data) && !bytes.Contains(data, []byte{0})
}
