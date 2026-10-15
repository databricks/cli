package dresources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/cli/bundle/direct/internal/retries"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/utils"
)

const postgresUpdateTimeout = 20 * time.Minute

// updatePostgres sends PATCH /api/2.0/postgres/{name}?update_mask=... and waits
// for the resulting operation, decoding its response into T.
//
// The API takes oneof members in update_mask under their group name (e.g.
// "spec.expiration" for ttl/expire_time/no_expiry), which sdk-go's FieldMask
// rejects as an unknown field. Such updates are therefore sent without sdk-go.
func updatePostgres[T any](ctx context.Context, client *sdkclients.Clients, name, updateMask string, body any) (*T, error) {
	api, err := apiclient.New(client.Config)
	if err != nil {
		return nil, err
	}
	var op postgres.Operation
	err = api.Do(ctx, http.MethodPatch, "/api/2.0/postgres/"+name, nil, map[string]any{"update_mask": updateMask}, body, &op)
	if err != nil {
		return nil, err
	}
	opName := utils.Deref(op.Name)
	if opName == "" {
		return nil, errors.New("invalid operation response: missing name")
	}
	return retries.Poll(ctx, postgresUpdateTimeout, func() (*T, *retries.Err) {
		if !utils.Deref(op.Done) {
			op = postgres.Operation{}
			if err := api.Do(ctx, http.MethodGet, "/api/2.0/postgres/"+opName, nil, nil, nil, &op); err != nil {
				return nil, retries.Halt(err)
			}
			if !utils.Deref(op.Done) {
				return nil, retries.Continues("operation " + opName + " is still running")
			}
		}
		if op.Error != nil {
			return nil, retries.Halt(fmt.Errorf("operation failed: %s", utils.Deref(op.Error.Message)))
		}
		if len(op.Response) == 0 {
			return nil, retries.Halt(errors.New("operation completed without a response"))
		}
		var result T
		if err := json.Unmarshal(op.Response, &result); err != nil {
			return nil, retries.Halt(fmt.Errorf("decode operation response: %w", err))
		}
		return &result, nil
	})
}
