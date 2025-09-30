package configure

import (
	"errors"
	"net/url"
)

func validateHost(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	if u.Host == "" || u.Scheme != "https" {
		return errors.New("must start with https://")
	}
	if u.Path != "" && u.Path != "/" {
		return errors.New("must use empty path")
	}
	return nil
}
