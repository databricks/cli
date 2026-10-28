package testserver

import "github.com/databricks/cli/libs/fault"

// faultEndpointHandler handles POST /__testserver/fault.
func faultEndpointHandler(fr *fault.Rules) HandlerFunc {
	return func(req Request) any {
		if err := fr.Register(req.Token, req.Body); err != nil {
			return Response{StatusCode: 400, Body: map[string]string{"error": err.Error()}}
		}
		return Response{StatusCode: 200}
	}
}
