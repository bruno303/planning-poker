package clientcollection

import (
	"planning-poker/internal/domain/entity"
	"testing"
)

func newTestClient(id, name string) *entity.Client {
	return &entity.Client{ID: id, Name: name}
}

func TestAddAndValues(t *testing.T) {
	cc := New()
	client := newTestClient("1", "Alice")
	cc.Add(client)

	values := cc.Values()
	if len(values) != 1 || values[0] != client {
		t.Errorf("expected Values to return the added client")
	}
}

func TestRemove(t *testing.T) {
	client1 := newTestClient("1", "Alice")
	client2 := newTestClient("2", "Bob")
	cc := New(client1, client2)

	cc.Remove("1")
	if cc.Count() != 1 || cc.Values()[0] != client2 {
		t.Errorf("expected only client2 to remain after removal")
	}

	cc.Remove("non-existent")
	if cc.Count() != 1 {
		t.Errorf("removing non-existent client should not change collection")
	}
}

func TestCount(t *testing.T) {
	cc := New()
	if cc.Count() != 0 {
		t.Errorf("expected count 0, got %d", cc.Count())
	}
	cc.Add(newTestClient("1", "Alice"))
	if cc.Count() != 1 {
		t.Errorf("expected count 1, got %d", cc.Count())
	}
}

func TestFirst(t *testing.T) {
	cc := New()
	if c, ok := cc.First(); ok || c != nil {
		t.Errorf("expected First to return nil, false for empty collection")
	}
	client := newTestClient("1", "Alice")
	cc.Add(client)
	if c, ok := cc.First(); !ok || c != client {
		t.Errorf("expected First to return the first client")
	}
}

func TestForEach(t *testing.T) {
	client1 := newTestClient("1", "Alice")
	client2 := newTestClient("2", "Bob")
	cc := New(client1, client2)

	var ids []string
	cc.ForEach(func(c *entity.Client) {
		ids = append(ids, c.ID)
	})
	if len(ids) != 2 || ids[0] != "1" || ids[1] != "2" {
		t.Errorf("ForEach did not iterate correctly")
	}
}

func TestFilter(t *testing.T) {
	client1 := newTestClient("1", "Alice")
	client2 := newTestClient("2", "Bob")
	cc := New(client1, client2)

	filtered := cc.Filter(func(c *entity.Client) bool {
		return c.Name == "Bob"
	})

	if filtered.Count() != 1 || filtered.Values()[0] != client2 {
		t.Errorf("Filter did not return the expected client")
	}
}
