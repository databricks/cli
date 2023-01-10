package repl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/commands"
)

const fileName = "repl-%s.json"

type Repl struct {
	ClusterID string
	ContextID string
	Language  commands.Language

	api *commands.CommandExecutionAPI
}

func (r *Repl) Isset() bool {
	return r.ClusterID != "" && r.ContextID != ""
}

func (r *Repl) IsRunning(ctx context.Context) (bool, error) {
	if !r.Isset() {
		return false, nil
	}

	res, err := r.api.ContextStatus(ctx, commands.ContextStatusRequest{
		ClusterId: r.ClusterID,
		ContextId: r.ContextID,
	})
	if err != nil {
		var aerr apierr.APIError

		// We can deal with API errors; bail if it is different.
		if !errors.As(err, &aerr) {
			return false, err
		}

		// The API returns a 500 if a context is missing; bail if it is different.
		if aerr.StatusCode != http.StatusInternalServerError {
			return false, err
		}

		// The context isn't running if the error message contains "ContextNotFound".
		if strings.HasPrefix(aerr.Message, "ContextNotFound:") {
			return false, nil
		}

		return false, err
	}

	switch res.Status {
	case commands.ContextStatusError:
		return false, nil
	case commands.ContextStatusPending:
		return false, nil
	case commands.ContextStatusRunning:
		return true, nil
	default:
		return false, fmt.Errorf("unknown status: %s", res.Status)
	}
}

func (r *Repl) Create(ctx context.Context) error {
	res, err := r.api.CreateAndWait(ctx, commands.CreateContext{
		ClusterId: r.ClusterID,
		Language:  r.Language,
	})
	if err != nil {
		return err
	}

	r.ContextID = res.Id
	return nil
}

func (r *Repl) Execute(ctx context.Context, code []byte) (*commands.CommandStatusResponse, error) {
	res, err := r.api.ExecuteAndWait(ctx, commands.Command{
		ClusterId: r.ClusterID,
		ContextId: r.ContextID,
		Language:  r.Language,
		Command:   string(code),
	})

	return res, err
}

func New(b *bundle.Bundle, lang commands.Language) (*Repl, error) {
	client, err := client.New(b.WorkspaceClient().Config)
	if err != nil {
		return nil, err
	}

	return &Repl{
		Language: lang,
		api:      commands.NewCommandExecution(client),
	}, nil
}

func (r *Repl) Load(b *bundle.Bundle) error {
	cacheDir, err := b.CacheDir()
	if err != nil {
		return err
	}

	path := filepath.Join(cacheDir, fmt.Sprintf(fileName, r.Language))
	buf, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if len(buf) > 0 {
		err = json.Unmarshal(buf, r)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repl) Save(b *bundle.Bundle) error {
	cacheDir, err := b.CacheDir()
	if err != nil {
		return err
	}
	path := filepath.Join(cacheDir, fmt.Sprintf(fileName, r.Language))
	buf, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return os.WriteFile(path, buf, 0700)

}

func Create(ctx context.Context, b *bundle.Bundle, lang commands.Language) (*Repl, error) {
	repl, err := New(b, lang)
	if err != nil {
		return nil, err
	}

	repl.ClusterID = b.Config.Bundle.DefaultCluster
	err = repl.Create(ctx)
	if err != nil {
		return nil, err
	}

	err = repl.Save(b)
	if err != nil {
		return nil, err
	}

	return repl, nil
}

func GetOrCreate(ctx context.Context, b *bundle.Bundle, lang commands.Language) (*Repl, error) {
	repl, err := New(b, lang)
	if err != nil {
		return nil, err
	}

	err = repl.Load(b)
	if err != nil {
		return nil, err
	}

	// If the version we loaded uses a different cluster ID, invalidate it.
	if b.Config.Bundle.DefaultCluster != repl.ClusterID {
		return Create(ctx, b, lang)
	}

	// If the repl is still running, return it.
	ok, err := repl.IsRunning(ctx)
	if err != nil {
		return nil, err
	}
	if ok {
		return repl, nil
	}

	// Otherwise, create a new one.
	return Create(ctx, b, lang)
}
