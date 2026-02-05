package clientcollection

import (
	"planning-poker/internal/domain/entity"
)

type InMemoryClientCollection struct {
	clients []*entity.Client
}

func New(clients ...*entity.Client) *InMemoryClientCollection {
	return &InMemoryClientCollection{
		clients: clients,
	}
}

func (cc *InMemoryClientCollection) Add(client *entity.Client) {
	cc.clients = append(cc.clients, client)
}

func (cc *InMemoryClientCollection) Remove(clientID string) {
	for i, c := range cc.clients {
		if c.ID == clientID {
			cc.clients = append(cc.clients[:i], cc.clients[i+1:]...)
			return
		}
	}
}

func (cc *InMemoryClientCollection) Values() []*entity.Client {
	return cc.clients
}

func (cc *InMemoryClientCollection) Count() int {
	return len(cc.clients)
}

func (cc *InMemoryClientCollection) First() (*entity.Client, bool) {
	if len(cc.clients) == 0 {
		return nil, false
	}
	return cc.clients[0], true
}

func (cc *InMemoryClientCollection) ForEach(f func(client *entity.Client)) {
	for _, client := range cc.clients {
		f(client)
	}
}

func (cc *InMemoryClientCollection) Filter(f func(client *entity.Client) bool) entity.ClientCollection {
	filtered := New()
	for _, client := range cc.clients {
		if f(client) {
			filtered.Add(client)
		}
	}
	return filtered
}
