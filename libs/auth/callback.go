package auth

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed page.tmpl
var pageTmpl string

type oauthResult struct {
	Error            string
	ErrorDescription string
	Host             string
	State            string
	Code             string
}

type callbackServer struct {
	ln          net.Listener
	srv         http.Server
	ctx         context.Context
	a           *PersistentAuth
	renderErrCh chan error
	feedbackCh  chan oauthResult
	tmpl        *template.Template
}

func newCallback(ctx context.Context, a *PersistentAuth) (*callbackServer, error) {
	tmpl, err := template.New("page").Funcs(template.FuncMap{
		"title": func(in string) string {
			title := cases.Title(language.English)
			return title.String(strings.ReplaceAll(in, "_", " "))
		},
	}).Parse(pageTmpl)
	if err != nil {
		return nil, err
	}
	cb := &callbackServer{
		feedbackCh:  make(chan oauthResult),
		renderErrCh: make(chan error),
		tmpl:        tmpl,
		ctx:         ctx,
		ln:          a.ln,
		a:           a,
	}
	cb.srv.Handler = cb
	go cb.srv.Serve(cb.ln)
	return cb, nil
}

func (cb *callbackServer) Close() error {
	return cb.srv.Close()
}

// ServeHTTP renders page.html template
func (cb *callbackServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := oauthResult{
		Error:            r.FormValue("error"),
		ErrorDescription: r.FormValue("error_description"),
		Code:             r.FormValue("code"),
		State:            r.FormValue("state"),
		Host:             cb.a.Host,
	}
	if res.Error != "" {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	err := cb.tmpl.Execute(w, res)
	if err != nil {
		cb.renderErrCh <- err
	}
	cb.feedbackCh <- res
}

// Handler opens up a browser waits for redirect to come back from the identity provider
func (cb *callbackServer) Handler(authCodeURL string) (string, string, error) {
	err := cb.a.browser(authCodeURL)
	if err != nil {
		fmt.Printf("Please open %s in the browser to continue authentication", authCodeURL)
	}
	select {
	case <-cb.ctx.Done():
		return "", "", cb.ctx.Err()
	case renderErr := <-cb.renderErrCh:
		return "", "", renderErr
	case res := <-cb.feedbackCh:
		if res.Error != "" {
			return "", "", fmt.Errorf("%s: %s", res.Error, res.ErrorDescription)
		}
		return res.Code, res.State, nil
	}
}
