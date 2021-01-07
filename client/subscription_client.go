package client

import (
	"encoding/json"

	"github.com/hasura/go-graphql-client"
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

func (sc *SubscriptionClient) Subscribe(callback func(msg *json.RawMessage, err error) error, query interface{}, vars map[string]interface{}) (string, error) {
	subID, err := sc.cl.Subscribe(query, nil, callback)

	// Subscriptions are lazily started
	go sc.cl.Run()

	return subID, err
}

func (sc *SubscriptionClient) Unsubscribe(subID string) {
	sc.cl.Unsubscribe(subID)
	sc.cl.Close()
}
