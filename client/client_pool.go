package client

import (
	"net"
	"net/http"
	"net/url"
)

// -----------------------------------------------------------------------------

type ClientPool interface {
	GetHost() string
	GetEndpoint() string
	GetClient() (*http.Client, string)
	Refresh(reason string) error
}

// -----------------------------------------------------------------------------

type defaultClientPool struct {
	client   *http.Client
	endpoint *url.URL
	host     string
}

func NewDefaultClientPool(endpoint *url.URL) (ClientPool, error) {
	host, _, err := net.SplitHostPort(endpoint.Host)
	if err != nil {
		return nil, err
	}

	return &defaultClientPool{
		client: http.DefaultClient,
		host:   host,
		endpoint: endpoint,
	}, nil
}

func (self *defaultClientPool) GetHost() string {
	return self.host
}

func (self *defaultClientPool) GetEndpoint() string {
	return self.endpoint.String()
}

func (self *defaultClientPool) GetClient() (*http.Client, string) {
	return self.client, self.host
}

func (self *defaultClientPool) Refresh(_ string) error {
	return nil
}
