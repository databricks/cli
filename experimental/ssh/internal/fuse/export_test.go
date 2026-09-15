package fuse

import "net/http"

var ParseRegistration = parseRegistration

const RefreshInterval = refreshInterval

func ConfigureTestClient(c *Client, httpClient *http.Client, hosts []string, readPID func() (int, error)) {
	c.http = httpClient
	c.hosts = hosts
	c.readPID = readPID
}

func HTTPClient(c *Client) *http.Client {
	return c.http
}
