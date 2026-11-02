package entity

// ClientID identifies a client inside a room.
type ClientID string

func (id ClientID) String() string {
	return string(id)
}

// StoryID identifies a story inside a room backlog.
type StoryID string

func (id StoryID) String() string {
	return string(id)
}
