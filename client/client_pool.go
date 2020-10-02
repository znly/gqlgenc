package client

import (
	"net/http"
)

// -----------------------------------------------------------------------------

type ClientPool interface {
	GetHost() string
	GetClient() (*http.Client, string)
	Refresh() error
}

// -----------------------------------------------------------------------------

type defaultClientPool struct {
	client *http.Client
	host   string
}

func NewDefaultClientPool(host string) ClientPool {
	return &defaultClientPool{
		client: http.DefaultClient,
		host:   host,
	}
}

func (self *defaultClientPool) GetHost() string {
	return self.host
}

func (self *defaultClientPool) GetClient() (*http.Client, string) {
	return self.client, self.host
}

func (self *defaultClientPool) Refresh() error {
	return nil
}
