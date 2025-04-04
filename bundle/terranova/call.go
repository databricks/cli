package terranova

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/httpclient"
)

type HTTPClient interface {
	MakeHTTPCall(ctx context.Context, method, path, requestBody string, response *any) error
}

type SDKHTTPClient struct {
	Client *httpclient.ApiClient
}

func (c SDKHTTPClient) MakeHTTPCall(ctx context.Context, method, path, requestBody string, response *any) error {
	opts := []httpclient.DoOption{httpclient.WithRequestData(requestBody)}
	if response != nil {
		opts = append(opts, httpclient.WithResponseUnmarshal(response))
	}
	return c.Client.Do(ctx, method, path, opts...)
}

type CallSpec struct {
	Method string

	// HTTP Path. Can encode resource as {}
	Path string

	// If present, add one pair to request dictionary: key=RequestIDField and value=ResourceID
	RequestIDField string

	// Same but resulting type is integer
	RequestIDIntegerField string

	// If present, a query parameter will be added to request with key=QueryIDField and value=ResourceID
	QueryIDField string

	// If present, response will be parsed as JSON dictionary and value from key=ResponseIDField will be extracted as ResourceID
	ResponseIDField string

	// If present, request data will be put under this field (instead of top level)
	RequestDataField string

	// If present, response data will be extract from this field (instead of top level)
	// ResponseDataField string

	// Additional processors to apply to request
	RequestProcessors []Processor

	// Additional processors to apply to response
	ResponseProcessors []Processor
}

type Call struct {
	Spec         *CallSpec
	Path         string
	RequestBody  string
	ResponseBody any
	StatusCode   int
	ResponseID   string
}

func (spec *CallSpec) PrepareCall(request dyn.Value, resourceID string) (*Call, error) {
	call := Call{
		Spec: spec,
	}

	call.Path = spec.Path
	if resourceID != "" {
		call.Path = strings.ReplaceAll(spec.Path, "{}", resourceID)

		var resourceIDConverted any
		var idfield string
		var err error

		if spec.RequestIDIntegerField != "" {
			resourceIDConverted, err = strconv.Atoi(resourceID)
			if err != nil {
				return nil, fmt.Errorf("Cannot convert resourceID to integer: %#v: %w", resourceID, err)
			}
			idfield = spec.RequestIDIntegerField
		} else {
			// keep string
			resourceIDConverted = resourceID
			idfield = spec.RequestIDField
		}

		if idfield != "" {
			// If we have a request body, we need to unmarshal it, add the ID field, and marshal it back
			var requestMap dyn.Mapping
			switch request.Kind() {
			case dyn.KindNil:
				requestMap = dyn.NewMapping()
			case dyn.KindMap:
				requestMap = request.MustMap()
				// good
			default:
				return nil, fmt.Errorf("Unexpected request type: %s", request.Kind().String())
			}

			requestMap.SetLoc(idfield, nil, dyn.V(resourceIDConverted))
			request = dyn.V(requestMap)
		}

		if spec.QueryIDField != "" {
			call.Path += fmt.Sprintf("?%s=%s", spec.QueryIDField, queryParamEscape(resourceID))
		}
	} else {
		if strings.Contains(spec.Path, "{}") {
			// Must fail, because it's not a valid path
			return nil, fmt.Errorf("CallSpec error: Path has {} but resourceID is not provided: %s", spec.Path)
		}
	}

	var err error
	request, err = ApplyProcessors(spec.RequestProcessors, request)
	if err != nil {
		return nil, err
	}

	requestBodyBytes, err := json.MarshalIndent(request.AsAny(), "", "  ")
	if err != nil {
		return nil, err
	}

	call.RequestBody = string(requestBodyBytes)

	return &call, nil
}

func queryParamEscape(s string) string {
	s = url.QueryEscape(s)
	s = strings.ReplaceAll(s, "+", "%20")
	return s
}

func (c *Call) Perform(ctx context.Context, apiclient HTTPClient) error {
	err := apiclient.MakeHTTPCall(ctx, c.Spec.Method, c.Path, c.RequestBody, &c.ResponseBody)

	if err == nil {
		c.StatusCode = 200
	} else {
		var apiErr *apierr.APIError
		if errors.As(err, &apiErr) {
			c.StatusCode = apiErr.StatusCode
		}
	}

	// Extract ResponseID from response body if field is specified
	if c.Spec.ResponseIDField != "" {
		respMap, ok := c.ResponseBody.(map[string]any)
		if ok {
			if id, ok := respMap[c.Spec.ResponseIDField]; ok {
				switch value := id.(type) {
				case string:
					c.ResponseID = value
				case int:
					c.ResponseID = strconv.Itoa(value)
				case float64:
					// XXX information was lost, fix decoder
					c.ResponseID = strconv.Itoa(int(value))
				default:
					return fmt.Errorf("Found ResponseIDField=%s in the response but type is unexpected: %T\nResponse: %s", c.Spec.ResponseIDField, value, c.ResponseBody)
				}
			}
		}
	}

	if err == nil {
		log.Infof(ctx, "Successful call %s %s: %d", c.Spec.Method, c.Path, c.StatusCode)
	} else {
		log.Warnf(ctx, "Failed call %s %s: %d %s", c.Spec.Method, c.Path, c.StatusCode, err.Error())
	}

	return err
}
