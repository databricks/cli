package appproxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type Proxy struct {
	ctx            context.Context
	targetURL      *url.URL
	client         *http.Client
	server         *http.Server
	headerToInject map[string]string
}

// Creates a new proxy instance that will forward all requests to the targetURL
// The targetURL should be a valid URL with a scheme and a host.
func New(targetURL string) (*Proxy, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	proxy := Proxy{targetURL: u, client: &http.Client{}, ctx: context.Background()}
	server := &http.Server{}
	server.Handler = http.HandlerFunc(proxy.proxyHandler)
	proxy.server = server
	return &proxy, nil
}

// Start starts the proxy server on the given address (host:port, e.g. localhost:8080)
// The proxy will forward all requests to the targetURL
func (p *Proxy) Listen(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

func (p *Proxy) Serve(ln net.Listener) error {
	return p.server.Serve(ln)
}

func (p *Proxy) Start(addr string) error {
	ln, err := p.Listen(addr)
	if err != nil {
		return err
	}
	return p.Serve(ln)
}

func (p *Proxy) Stop() error {
	return p.server.Shutdown(p.ctx)
}

// InjectHeader injects a header that will be added to all requests forwarded by the proxy
func (p *Proxy) InjectHeader(key, value string) {
	if p.headerToInject == nil {
		p.headerToInject = make(map[string]string)
	}
	p.headerToInject[key] = value
}

func (p *Proxy) proxyHandler(w http.ResponseWriter, r *http.Request) {
	if p.isWebSocketRequest(r) {
		p.handleWebSocket(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

func (p *Proxy) isWebSocketRequest(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

func (p *Proxy) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Connect to the running websocket server to which we will proxy the request
	targetServerConn, err := net.Dial("tcp", p.targetURL.Host)
	if err != nil {
		http.Error(w, "Error connecting to backend server", http.StatusInternalServerError)
		return
	}
	defer targetServerConn.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	// We need to hijack the connection to be able to proxy the request
	// to the websocket server. We can use the hijacked connection to
	// read and write messages back and forth from the client to the app.
	middlewareConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, "Hijacking failed", http.StatusInternalServerError)
		return
	}
	defer middlewareConn.Close()

	err = r.Write(targetServerConn)
	if err != nil {
		return
	}

	// Start proxying data between the client and the server
	errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errc <- err
	}

	// Copy request messages from client to server
	go cp(targetServerConn, middlewareConn)

	// Copy response messages from server to client
	go cp(middlewareConn, targetServerConn)

	err = <-errc
	if err != nil {
		// If the error is not EOF, then there was a problem
		if err != io.EOF {
			http.Error(w, "Error copying messages", http.StatusInternalServerError)
		}
	}
}

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	r.RequestURI = ""
	r.URL.Scheme = p.targetURL.Scheme
	r.URL.Host = p.targetURL.Host
	r.Host = p.targetURL.Host

	// Inject additional headers
	for k, v := range p.headerToInject {
		r.Header.Add(k, v)
	}

	resp, err := p.client.Do(r)
	if err != nil {
		http.Error(w, "Error forwarding request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, "Error reading a response", http.StatusInternalServerError)
		return
	}
}
