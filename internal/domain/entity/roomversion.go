package entity

import "strconv"

// RoomVersion is the optimistic-concurrency revision of a room. Domain
// mutations never bump it; persistence adapters set it after a successful save.
type RoomVersion uint64

// NewRoomVersion builds a RoomVersion from its persisted numeric form.
func NewRoomVersion(v uint64) RoomVersion {
	return RoomVersion(v)
}

// Uint64 returns the version in its persisted numeric form.
func (v RoomVersion) Uint64() uint64 {
	return uint64(v)
}

// Next returns the version that follows v.
func (v RoomVersion) Next() RoomVersion {
	return v + 1
}

// String renders the version as a decimal number.
func (v RoomVersion) String() string {
	return strconv.FormatUint(uint64(v), 10)
}
