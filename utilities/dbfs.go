package utilities

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/dbfs"
	"github.com/databricks/databricks-sdk-go/workspaces"
)

// move to go sdk / replace with utility function once
// https://github.com/databricks/databricks-sdk-go/issues/57 is Done
// Tracked in https://github.com/databricks/bricks/issues/25
func CreateDbfsFile(ctx context.Context,
	wsc *workspaces.WorkspacesClient,
	path string,
	contents []byte,
	overwrite bool,
) error {
	createResponse, err := wsc.Dbfs.Create(ctx,
		dbfs.CreateRequest{
			Overwrite: overwrite,
			Path:      path,
		},
	)
	if err != nil {
		return err
	}
	handle := createResponse.Handle
	buffer := bytes.NewBuffer(contents)
	for {
		byteChunk := buffer.Next(1e6)
		if len(byteChunk) == 0 {
			break
		}
		b64Data := base64.StdEncoding.EncodeToString(byteChunk)
		err := wsc.Dbfs.AddBlock(ctx,
			dbfs.AddBlockRequest{
				Data:   b64Data,
				Handle: handle,
			},
		)
		if err != nil {
			// TODO: Add some better error reporting here
			return err
		}
	}
	return nil
}

func ReadDbfsFile(ctx context.Context,
	wsc *workspaces.WorkspacesClient,
	path string,
) (content []byte, err error) {
	fetchLoop := true
	offSet := 0
	length := int(1e6)
	for fetchLoop {
		dbfsReadReponse, err := wsc.Dbfs.Read(ctx,
			dbfs.ReadRequest{
				Path:   path,
				Offset: offSet,
				Length: length,
			},
		)
		if err != nil {
			return content, fmt.Errorf("cannot read %s: %w", path, err)
		}
		if dbfsReadReponse.BytesRead == 0 || dbfsReadReponse.BytesRead < int64(length) {
			fetchLoop = false
		}
		decodedBytes, err := base64.StdEncoding.DecodeString(dbfsReadReponse.Data)
		if err != nil {
			return content, err
		}
		content = append(content, decodedBytes...)
		offSet += length
	}
	return content, err
}
