package planningpoker

type Room struct {
	ID      string
	Clients map[string]*Client
}

type Hub struct {
	Rooms map[string]*Room
}

func NewHub() *Hub {
	return &Hub{
		Rooms: make(map[string]*Room),
	}
}

func (h *Hub) GetRoom(roomID string) *Room {
	if room, exists := h.Rooms[roomID]; exists {
		return room
	}
	room := &Room{
		ID:      roomID,
		Clients: make(map[string]*Client),
	}
	h.Rooms[roomID] = room
	return room
}

func (h *Hub) RemoveRoom(roomID string) {
	delete(h.Rooms, roomID)
}

func (r *Room) AddClient(client *Client) {
	r.Clients[client.ID] = client
}

func (r *Room) RemoveClient(clientID string) {
	delete(r.Clients, clientID)
	if len(r.Clients) == 0 {
		// Optionally, you can implement logic to remove the room from the hub if empty
	}
}

func (r *Room) Broadcast(message any) {
	for _, client := range r.Clients {
		client.Send(message)
	}
}
