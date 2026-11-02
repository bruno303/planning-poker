package entity

// Vote is a single participant vote label, expressed in the room deck's
// vocabulary.
type Vote string

// String returns the vote label.
func (v Vote) String() string {
	return string(v)
}

// IsEmpty reports whether the vote carries no label.
func (v Vote) IsEmpty() bool {
	return v == ""
}

// VotePtr wraps a label into an optional vote.
func VotePtr(v string) *Vote {
	vote := Vote(v)
	return &vote
}

// Ptr returns the underlying label, preserving nil.
func (v *Vote) Ptr() *string {
	if v == nil {
		return nil
	}
	value := string(*v)
	return &value
}
