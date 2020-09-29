package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/Yamashou/gqlgenc/graphqljson"
	"golang.org/x/xerrors"
)

type HTTPRequestOption func(ctx context.Context, req *http.Request)
type HTTPResponseCallback func(ctx context.Context, res *http.Response)

type Client struct {
	Client                *http.Client
	Endpoint              string
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
	client *http.Client, endpoint string,
	options []HTTPRequestOption,
	callbacks []HTTPResponseCallback,
) *Client {
	return &Client{
		Client:                client,
		Endpoint:              endpoint,
		HTTPRequestOptions:    options,
		HTTPResponseCallbacks: callbacks,
	}
}

func (c *Client) newRequest(
	ctx context.Context,
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, xerrors.Errorf("create request struct failed: %w", err)
	}

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
	req, err := c.newRequest(ctx, query, vars, httpRequestOptions, httpResponseCallbacks)
	if err != nil {
		return xerrors.Errorf("don't create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	res, err := c.Client.Do(req)
	if err != nil {
		return xerrors.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if err := graphqljson.Unmarshal(res.Body, respData); err != nil {
		return err
	}

	if res.StatusCode < 200 || 299 < res.StatusCode {
		return xerrors.Errorf("http status code: %v", res.StatusCode)
	}

	for _, callback := range httpResponseCallbacks {
		callback(ctx, res)
	}

	return nil
}
