package client

import (
	"encoding/json"

	graphql "github.com/hasura/go-graphql-client"
)

type SubscriptionClient struct {
	cl *graphql.SubscriptionClient
}

// NewSubscriptionClient instanciates a new GraphQL subscription client based on the Websockets protocol
func NewSubscriptionClient(wsURI string) *SubscriptionClient {
	cl := graphql.NewSubscriptionClient(wsURI)

	return &SubscriptionClient{
		cl,
	}
}

func (sc *SubscriptionClient) Subscribe(out chan<- interface{}, query string, vars map[string]interface{}) (string, error) {
	subID, err := sc.cl.Subscribe(query, nil, func(msg *json.RawMessage, err error) error {
		out <- msg
		return err
	})

	// Subscriptions are lazily started
	sc.cl.Run()

	return subID, err
}

func (sc *SubscriptionClient) Unsubscribe(subID string) {
	sc.cl.Unsubscribe(subID)
	sc.cl.Close()
}
