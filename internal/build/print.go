package build

import (
	"fmt"
	"io"
)

func PrintVersion(w io.Writer) error {
	_, err := fmt.Fprintf(w, "Databricks CLI v%s\n", GetInfo().Version)
	return err
}
