package auth

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/browser"
)

//go:embed page.tmpl
var pageTmpl string

const appRedirectAddr = "localhost:8020"

type oauthResult struct {
	Error            string
	ErrorDescription string
	Host             string
	State            string
	Code             string
}

type callbackServer struct {
	ln       net.Listener
	srv      *http.Server
	ctx      context.Context
	a        *PersistentAuth
	errc     chan error
	feedback chan oauthResult
	tmpl     *template.Template
	browser  func(string) error
}

func newCallback(ctx context.Context, a *PersistentAuth) (*callbackServer, error) {
	tmpl, err := template.New("page").Funcs(template.FuncMap{
		"title": func(in string) string {
			// TODO: use x/text/cases
			return strings.Title(strings.ReplaceAll(in, "_", " "))
		},
	}).Parse(pageTmpl)
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", appRedirectAddr)
	if err != nil {
		return nil, fmt.Errorf("listener: %w", err)
	}
	cb := &callbackServer{
		feedback: make(chan oauthResult),
		errc:     make(chan error),
		browser:  browser.OpenURL,
		srv:      &http.Server{},
		tmpl:     tmpl,
		ctx:      ctx,
		ln:       ln,
		a:        a,
	}
	cb.srv.Handler = cb
	go cb.srv.Serve(cb.ln)
	return cb, nil
}

func (cb *callbackServer) Close() error {
	return cb.srv.Close()
}

func (cb *callbackServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := oauthResult{
		Error:            r.FormValue("error"),
		ErrorDescription: r.FormValue("error_description"),
		Code:             r.FormValue("code"),
		State:            r.FormValue("state"),
		Host:             cb.a.Host,
	}
	w.WriteHeader(200)
	err := cb.tmpl.Execute(w, res)
	if err != nil {
		cb.errc <- err
	}
	cb.feedback <- res
}

func (cb *callbackServer) AuthorizationHandler(authCodeURL string) (string, string, error) {
	err := cb.browser(authCodeURL)
	if err != nil {
		return "", "", fmt.Errorf("cannot open browser: %w", err)
	}
	ctx, cancel := context.WithTimeout(cb.ctx, 5*time.Minute)
	defer cancel()
	select {
	case <-ctx.Done():
		return "", "", fmt.Errorf("timed out")
	case renderErr := <-cb.errc:
		return "", "", renderErr
	case res := <-cb.feedback:
		return res.Code, res.State, nil
	}
}
