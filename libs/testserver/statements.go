package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/databricks/cli/libs/testserver/testsql"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

// HandleSQL registers a matcher that runs fn when a submitted statement equals
// statement exactly (after trimming).
func (s *Server) HandleSQL(statement string, fn func(testsql.Request) testsql.Result) {
	s.sqlHandler.Handle(statement, fn)
}

// HandleSQLPattern registers a matcher that runs fn when re matches a submitted
// statement.
func (s *Server) HandleSQLPattern(re *regexp.Regexp, fn func(testsql.Request) testsql.Result) {
	s.sqlHandler.HandlePattern(re, fn)
}

// sqlExecuteStatement handles POST /api/2.0/sql/statements. A statement that
// terminates as FAILED comes back as HTTP 200 with state=FAILED; the engine
// builds that response and this HTTP layer is just transport.
func (s *Server) sqlExecuteStatement(req Request) any {
	var r sql.ExecuteStatementRequest
	if err := json.Unmarshal(req.Body, &r); err != nil {
		return Response{StatusCode: http.StatusBadRequest, Body: fmt.Sprintf("invalid execute statement request: %s", err)}
	}
	return s.sqlHandler.Submit(r.Statement, r.WaitTimeout, r.Parameters)
}

// sqlGetStatement handles GET /api/2.0/sql/statements/{statement_id}.
func (s *Server) sqlGetStatement(req Request) any {
	resp := s.sqlHandler.Get(req.Vars["statement_id"])
	if resp == nil {
		return Response{StatusCode: http.StatusNotFound}
	}
	return resp
}

// sqlGetStatementResultChunk handles GET
// /api/2.0/sql/statements/{statement_id}/result/chunks/{chunk_index}.
func (s *Server) sqlGetStatementResultChunk(req Request) any {
	idx, err := strconv.Atoi(req.Vars["chunk_index"])
	if err != nil {
		return Response{StatusCode: http.StatusBadRequest, Body: fmt.Sprintf("invalid chunk index: %s", err)}
	}
	data := s.sqlHandler.Chunk(req.Vars["statement_id"], idx)
	if data == nil {
		return Response{StatusCode: http.StatusNotFound}
	}
	return data
}

// sqlCancelStatement handles POST /api/2.0/sql/statements/{statement_id}/cancel.
func (s *Server) sqlCancelStatement(req Request) any {
	s.sqlHandler.Cancel(req.Vars["statement_id"])
	return map[string]any{}
}
