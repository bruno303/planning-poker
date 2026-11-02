package entity

// RoomVersion is a room revision used for optimistic concurrency control.
type RoomVersion uint64

// Uint64 returns the version as a plain number.
func (v RoomVersion) Uint64() uint64 {
	return uint64(v)
}

// Next returns the following room revision.
func (v RoomVersion) Next() RoomVersion {
	return v + 1
}
