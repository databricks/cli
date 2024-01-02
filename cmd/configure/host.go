package configure

import (
	"fmt"
	"net/url"
)

func validateHost(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	if u.Host == "" || u.Scheme != "https" {
		return fmt.Errorf("must start with https://")
	}
	if u.Path != "" && u.Path != "/" {
		return fmt.Errorf("must use empty path")
	}
	return nil
}
