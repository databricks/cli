// Package fuse registers the SSH server's credentials with the compute's filesystem daemons.
package fuse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/databricks/cli/libs/log"
)

const (
	commandOrigin  = "RemoteSshServer"
	requestTimeout = 2 * time.Second
)

type request struct {
	APIToken       string            `json:"apiToken"`
	ProcStartTime  uint64            `json:"procStartTime"`
	CommandOrigin  string            `json:"commandOrigin"`
	PIDNamespaceID uint32            `json:"namespaceId"`
	AdditionalTags map[string]string `json:"additionalTags,omitempty"`
}

type daemon struct {
	name  string
	port  int
	path  string
	host  string
	token string
}

// Client tracks the last successful registration for each daemon. It must not be used concurrently.
type Client struct {
	registration Registration
	hosts        []string
	daemons      []daemon
	http         *http.Client
	readPID      func() (int, error)
}

// NewClient creates a client for the node-local filesystem daemons.
func NewClient(r Registration) (*Client, error) {
	if r.PID <= 0 || r.PIDNamespaceID == 0 || r.StartTime == 0 {
		return nil, errors.New("FUSE registration requires a positive PID, PID namespace and process start time")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Credentials belong on this node, even when the server has HTTP_PROXY set.
	transport.Proxy = nil
	return &Client{
		registration: r,
		readPID:      registeredPID,
		hosts:        []string{"databricks.node.host.local", "node.host.local", "localhost"},
		daemons: []daemon{
			{name: "workspace files", port: 1021, path: fmt.Sprintf("/api/1/pid/%d", r.PID)},
			{name: "volumes", port: 1015, path: fmt.Sprintf("/dbfs-fuse-api/1/pid/%d", r.PID)},
		},
		http: &http.Client{
			Timeout:       requestTimeout,
			Transport:     transport,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

// Register updates changed credentials and retries failed registrations. Force also restores a lost registration.
func (c *Client) Register(ctx context.Context, token, userID string, force bool) error {
	if token == "" {
		return errors.New("cannot register an empty FUSE token")
	}
	body := request{
		APIToken:       token,
		ProcStartTime:  c.registration.StartTime,
		CommandOrigin:  commandOrigin,
		PIDNamespaceID: c.registration.PIDNamespaceID,
	}
	if userID != "" {
		body.AdditionalTags = map[string]string{"userId": userID}
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to encode FUSE registration: %w", err)
	}

	var errs []error
	for i := range c.daemons {
		d := &c.daemons[i]
		if force {
			d.token = ""
		}
		if d.token == token {
			continue
		}
		if err := c.registerDaemon(ctx, d, encoded); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", d.name, err))
			continue
		}
		d.token = token
		log.Infof(ctx, "Registered SSH server PID %d with %s", c.registration.PID, d.name)
	}
	return errors.Join(errs...)
}

func (c *Client) registerDaemon(ctx context.Context, d *daemon, body []byte) error {
	if d.host != "" {
		return c.put(ctx, d, d.host, body)
	}
	var errs []error
	for _, host := range c.hosts {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := c.put(ctx, d, host, body); err != nil {
			errs = append(errs, err)
			continue
		}
		// DNS resolution alone does not establish that this endpoint serves the daemon.
		d.host = host
		return nil
	}
	return errors.Join(errs...)
}

func (c *Client) put(ctx context.Context, d *daemon, host string, body []byte) error {
	url := "http://" + net.JoinHostPort(host, strconv.Itoa(d.port)) + d.path
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build FUSE request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register with %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Error bodies may echo credentials from the request.
		return fmt.Errorf("%s returned HTTP %d", url, resp.StatusCode)
	}
	return nil
}
