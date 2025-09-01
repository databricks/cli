package output

import "fmt"

type PipelineOutput struct {
	UpdateId string `json:"update_id"`
}

func (out *PipelineOutput) String() (string, error) {
	return fmt.Sprintf("Update ID: %s\n", out.UpdateId), nil
}
