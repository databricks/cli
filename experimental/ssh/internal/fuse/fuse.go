// Package fuse registers a process with the node-local FUSE daemons that serve
// /Workspace, /Volumes and /dbfs on Databricks compute.
//
// The daemons authorize a caller by its process ancestry. In an SSH tunnel session
// the only registered ancestor is the bootstrap notebook's Python REPL, so when
// that REPL exits the session loses /Workspace and /Volumes while /dbfs and the
// REST API keep working. Registering the server's own pid removes that dependency.
package fuse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// wsfsPort serves /Workspace; ucFusePort serves /Volumes and /dbfs.
	wsfsPort    = 1021
	wsfsPidPath = "/api/1/pid/%d"

	ucFusePort    = 1015
	ucFusePidPath = "/dbfs-fuse-api/1/pid/%d"

	// CommandOrigin labels the registration in the daemons' audit logs.
	CommandOrigin = "RemoteSshServer"

	// The daemons are node-local, so a slower request will not succeed by waiting.
	requestTimeout = 2 * time.Second
)

// hostCandidates are the names the daemons may be reachable under, most specific
// first.
var hostCandidates = []string{"databricks.node.host.local", "node.host.local", "localhost"}

// Registration identifies the process whose credentials are being set. All three
// fields are required: the start time is what keeps a recycled pid from inheriting
// the credentials of the process that previously held it.
type Registration struct {
	PID            int
	PidNamespaceID uint32
	StartTime      uint64
}

func (r Registration) String() string {
	return fmt.Sprintf("pid=%d ns=%d startTime=%d", r.PID, r.PidNamespaceID, r.StartTime)
}

// Self describes the calling process.
func Self() (Registration, error) {
	pid := os.Getpid()

	namespaceID, err := pidNamespaceID()
	if err != nil {
		return Registration{}, err
	}

	startTime, err := processStartTime(pid)
	if err != nil {
		return Registration{}, err
	}

	return Registration{PID: pid, PidNamespaceID: namespaceID, StartTime: startTime}, nil
}

// request is the body both daemons accept. An empty apiToken clears the
// registration, and a non-empty one requires a non-zero procStartTime.
type request struct {
	APIToken       string            `json:"apiToken"`
	ProcStartTime  uint64            `json:"procStartTime"`
	CommandOrigin  string            `json:"commandOrigin"`
	PidNamespaceID uint32            `json:"namespaceId"`
	AdditionalTags map[string]string `json:"additionalTags,omitempty"`
}

// Client talks to the two node-local FUSE daemons. The ports are fields rather
// than constants at the call site only so that tests can point the client at a
// pair of local servers.
type Client struct {
	host       string
	wsfsPort   int
	ucFusePort int
	http       *http.Client
}

// NewClient resolves the host the daemons are reachable under, once, because it
// cannot change while the process runs.
func NewClient() (*Client, error) {
	for _, host := range hostCandidates {
		if _, err := net.LookupHost(host); err == nil {
			return &Client{
				host:       host,
				wsfsPort:   wsfsPort,
				ucFusePort: ucFusePort,
				http:       &http.Client{Timeout: requestTimeout},
			}, nil
		}
	}
	return nil, fmt.Errorf("none of the FUSE daemon hosts resolved: %s", strings.Join(hostCandidates, ", "))
}

// Host is the name the client sends its requests to.
func (c *Client) Host() string {
	return c.host
}

// Register grants r's process, and its descendants, access to /Workspace,
// /Volumes and /dbfs under token. userID is recorded for audit logging and may be
// empty.
func (c *Client) Register(ctx context.Context, r Registration, token, userID string) error {
	if token == "" {
		return errors.New("refusing to register an empty token, which clears the registration instead")
	}
	if r.StartTime == 0 {
		return fmt.Errorf("refusing to register %s: a zero start time is rejected", r)
	}

	body := request{
		APIToken:       token,
		ProcStartTime:  r.StartTime,
		CommandOrigin:  CommandOrigin,
		PidNamespaceID: r.PidNamespaceID,
	}
	if userID != "" {
		body.AdditionalTags = map[string]string{"userId": userID}
	}
	return c.put(ctx, r.PID, body)
}

// Revoke clears r's registration.
func (c *Client) Revoke(ctx context.Context, r Registration) error {
	return c.put(ctx, r.PID, request{
		CommandOrigin:  CommandOrigin,
		PidNamespaceID: r.PidNamespaceID,
	})
}

// put sends body to both daemons and joins their errors: one of the two being
// registered is better than neither.
func (c *Client) put(ctx context.Context, pid int, body request) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to encode FUSE registration: %w", err)
	}

	wsfsErr := c.putOne(ctx, c.wsfsPort, fmt.Sprintf(wsfsPidPath, pid), encoded)
	if wsfsErr != nil {
		wsfsErr = fmt.Errorf("wsfs: %w", wsfsErr)
	}
	ucFuseErr := c.putOne(ctx, c.ucFusePort, fmt.Sprintf(ucFusePidPath, pid), encoded)
	if ucFuseErr != nil {
		ucFuseErr = fmt.Errorf("uc-fuse: %w", ucFuseErr)
	}
	return errors.Join(wsfsErr, ucFuseErr)
}

func (c *Client) putOne(ctx context.Context, port int, path string, body []byte) error {
	url := "http://" + net.JoinHostPort(c.host, strconv.Itoa(port)) + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build a request for %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		reason, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s returned %s: %s", url, resp.Status, bytes.TrimSpace(reason))
	}
	return nil
}

func pidNamespaceID() (uint32, error) {
	link, err := os.Readlink("/proc/self/ns/pid")
	if err != nil {
		return 0, fmt.Errorf("failed to read the pid namespace: %w", err)
	}
	return parsePidNamespace(link)
}

// parsePidNamespace reads the inode out of the "pid:[4026531836]" form that
// /proc/<pid>/ns/pid links to.
func parsePidNamespace(link string) (uint32, error) {
	inner, ok := strings.CutPrefix(strings.TrimSpace(link), "pid:[")
	if !ok {
		return 0, fmt.Errorf("unexpected pid namespace link %q", link)
	}
	inner, ok = strings.CutSuffix(inner, "]")
	if !ok {
		return 0, fmt.Errorf("unexpected pid namespace link %q", link)
	}
	id, err := strconv.ParseUint(inner, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the pid namespace id in %q: %w", link, err)
	}
	return uint32(id), nil
}

// processStartTime reads a process's start time in clock ticks since boot.
func processStartTime(pid int) (uint64, error) {
	content, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, fmt.Errorf("failed to read the stat of pid %d: %w", pid, err)
	}
	startTime, err := parseStatStartTime(string(content))
	if err != nil {
		return 0, fmt.Errorf("failed to parse the stat of pid %d: %w", pid, err)
	}
	return startTime, nil
}

// parseStatStartTime returns field 22 (starttime) of /proc/<pid>/stat. Field 2 is
// the executable name in parentheses and may itself contain whitespace and
// parentheses, so the fields are counted from the last ')' rather than from the
// start of the line.
func parseStatStartTime(content string) (uint64, error) {
	end := strings.LastIndex(content, ")")
	if end == -1 {
		return 0, fmt.Errorf("no comm field in %q", content)
	}
	const startTimeIndex = 19
	fields := strings.Fields(content[end+1:])
	if len(fields) <= startTimeIndex {
		return 0, fmt.Errorf("only %d fields after comm in %q", len(fields), content)
	}
	return strconv.ParseUint(fields[startTimeIndex], 10, 64)
}
