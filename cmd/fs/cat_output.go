package fs

type catOutput struct {
	Content string `json:"content"`
}

func toCatOutput(content string) *catOutput {
	return &catOutput{
		Content: content,
	}
}
