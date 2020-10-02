package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Yamashou/gqlgenc/graphqljson"
	"golang.org/x/xerrors"
)

type HTTPRequestOption func(ctx context.Context, req *http.Request)
type HTTPResponseCallback func(ctx context.Context, res *http.Response)

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

func (c *Client) Post(
	ctx context.Context,
	respData interface{},
	query string, vars map[string]interface{},
	httpRequestOptions []HTTPRequestOption,
	httpResponseCallbacks []HTTPResponseCallback,
) error {
	host := c.ClientPool.GetHost()
	httpCl, httpEndpoint := c.ClientPool.GetClient()

	fmt.Println(httpEndpoint)

	for {
		req, err := c.newRequest(ctx,
			host, httpEndpoint,
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
					c.ClientPool.Refresh()
					continue
				}
			}
			return xerrors.Errorf("request failed: %w", err)
		}

		if err := graphqljson.Unmarshal(res.Body, respData); err != nil {
			res.Body.Close()
			return err
		}
		res.Body.Close()

		if res.StatusCode < 200 || 299 < res.StatusCode {
			return xerrors.Errorf("http status code: %v", res.StatusCode)
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
