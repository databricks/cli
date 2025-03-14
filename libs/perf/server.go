package perf

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	_ "net/http/pprof"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
	"github.com/gorilla/mux"
)

type Body struct {
	Args []string `json:"args"`
}

func StartServer() *httptest.Server {
	router := mux.NewRouter()
	server := httptest.NewServer(router)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		parsed := Body{}

		err := json.NewDecoder(r.Body).Decode(&parsed)
		if err != nil {
			panic(err)
		}

		cli := cmd.New(context.Background())
		cli.SetArgs(parsed.Args)

		err = root.Execute(cli.Context(), cli)
		if err != nil {
			panic(err)
		}
	})

	return server
}
