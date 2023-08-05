package feature

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func httpCall(ctx context.Context, url string, response any) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("github request failed: %s", res.Status)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(raw, response)
	if err != nil {
		return err
	}
	return nil
}
