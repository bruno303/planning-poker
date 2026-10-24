package entity

// testCollection is a slice-backed ClientCollection used by in-package tests
// where the collection itself is not the behaviour under test.
type testCollection struct {
	clients []*Client
}

func newTestCollection(clients ...*Client) *testCollection {
	return &testCollection{
		clients: clients,
	}
}

func (cc *testCollection) Add(client *Client) {
	cc.clients = append(cc.clients, client)
}

func (cc *testCollection) Remove(clientID string) {
	for i, client := range cc.clients {
		if client.ID == clientID {
			cc.clients = append(cc.clients[:i], cc.clients[i+1:]...)
			return
		}
	}
}

func (cc *testCollection) Count() int {
	return len(cc.clients)
}

func (cc *testCollection) First() (*Client, bool) {
	if len(cc.clients) == 0 {
		return nil, false
	}

	return cc.clients[0], true
}

func (cc *testCollection) ForEach(f func(client *Client)) {
	for _, client := range cc.clients {
		f(client)
	}
}

func (cc *testCollection) Filter(f func(client *Client) bool) ClientCollection {
	filtered := newTestCollection()
	for _, client := range cc.clients {
		if f(client) {
			filtered.Add(client)
		}
	}

	return filtered
}

func (cc *testCollection) Values() []*Client {
	return cc.clients
}
