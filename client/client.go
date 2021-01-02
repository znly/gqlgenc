package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Yamashou/gqlgenc/graphqljson"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"golang.org/x/xerrors"
)

// HTTPRequestOption represents the options applicable to the http client
type HTTPRequestOption func(ctx context.Context, req *http.Request)
type HTTPResponseCallback func(ctx context.Context, res *http.Response)

// Client is the http client wrapper
type Client struct {
	ClientPool            ClientPool
	HTTPRequestOptions    []HTTPRequestOption
	HTTPResponseCallbacks []HTTPResponseCallback
}

// Request represents an outgoing GraphQL request
type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// NewClient creates a new http client wrapper
func NewClient(
	clientPool ClientPool,
	options []HTTPRequestOption,
	callbacks []HTTPResponseCallback,
) *Client {
	return &Client{
		ClientPool:            clientPool,
		HTTPRequestOptions:    options,
		HTTPResponseCallbacks: callbacks,
	}
}

func (c *Client) newRequest(
	ctx context.Context,
	host, endpoint string,
	query string, vars map[string]interface{},
	httpRequestOptions []HTTPRequestOption,
	httpResponseCallbacks []HTTPResponseCallback,
) (*http.Request, error) {
	r := &Request{
		Query:         query,
		Variables:     vars,
		OperationName: "",
	}

	requestBody, err := json.Marshal(r)
	if err != nil {
		return nil, xerrors.Errorf("encode: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, xerrors.Errorf("create request struct failed: %w", err)
	}
	req.Host = host

	for _, httpRequestOption := range c.HTTPRequestOptions {
		httpRequestOption(ctx, req)
	}
	for _, httpRequestOption := range httpRequestOptions {
		httpRequestOption(ctx, req)
	}

	return req, nil
}

// GqlErrorList is the struct of a standard graphql error response
type GqlErrorList struct {
	Errors gqlerror.List `json:"errors"`
}

func (e *GqlErrorList) Error() string {
	return e.Errors.Error()
}

// HTTPError is the error when a GqlErrorList cannot be parsed
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represent an handled error
type ErrorResponse struct {
	// populated when http status code is not OK
	NetworkError *HTTPError `json:"networkErrors"`
	// populated when http status code is OK but the server returned at least one graphql error
	GqlErrors *gqlerror.List `json:"graphqlErrors"`
}

// HasErrors returns true when at least one error is declared
func (er *ErrorResponse) HasErrors() bool {
	return er.NetworkError != nil || er.GqlErrors != nil
}

func (er *ErrorResponse) Error() string {
	content, err := json.Marshal(er)
	if err != nil {
		return err.Error()
	}

	return string(content)
}

// Post sends a http POST request to the graphql endpoint with the given query then unpacks
// the response into the given object.
func (c *Client) Post(
	ctx context.Context,
	respData interface{},
	query string, vars map[string]interface{},
	httpRequestOptions []HTTPRequestOption,
	httpResponseCallbacks []HTTPResponseCallback,
) error {
	host := c.ClientPool.GetHost()
	endpoint := c.ClientPool.GetEndpoint()

	for {
		httpCl, _ := c.ClientPool.GetClient()

		req, err := c.newRequest(ctx,
			host, endpoint,
			query, vars,
			httpRequestOptions, httpResponseCallbacks,
		)
		if err != nil {
			return xerrors.Errorf("don't create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Accept", "application/json; charset=utf-8")

		res, err := httpCl.Do(req)
		if err != nil {
			if innerErr, ok := err.(*url.Error); ok {
				if !(innerErr.Err == context.DeadlineExceeded ||
					innerErr.Err == context.Canceled) {
					c.ClientPool.Refresh(fmt.Sprintf("%#v (%#v)", err, innerErr.Err))
					continue
				}
			}
			return xerrors.Errorf("request failed: %w", err)
		}
		body, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return xerrors.Errorf("failed to read response body: %w", err)
		}

		if res.StatusCode/100 != 2 {
			return xerrors.Errorf("http status code: %v", res.StatusCode)
		}

		if err := unmarshal(body, respData); err != nil {
			return err
		}

		for _, httpResponseCallback := range c.HTTPResponseCallbacks {
			httpResponseCallback(ctx, res)
		}
		for _, callback := range httpResponseCallbacks {
			callback(ctx, res)
		}
		return nil
	}
}

// response is a GraphQL layer response from a handler.
type response struct {
	Data   json.RawMessage `json:"data"`
	Errors json.RawMessage `json:"errors"`
}

func unmarshal(data []byte, res interface{}) error {
	resp := response{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return xerrors.Errorf("failed to decode data %s: %w", string(data), err)
	}

	if len(resp.Errors) > 0 {
		// try to parse standard graphql error
		errors := &GqlErrorList{}
		if e := json.Unmarshal(data, errors); e != nil {
			return xerrors.Errorf("faild to parse graphql errors. Response content %s - %w ", string(data), e)
		}

		return errors
	}

	if err := graphqljson.UnmarshalData(resp.Data, res); err != nil {
		return xerrors.Errorf("failed to decode data into response %s: %w", string(data), err)
	}
	return nil
}
