package entity

// Vote is an immutable estimate cast by a client. Deck validation belongs to
// the Room, so an empty Vote is a valid "no estimate" value here.
type Vote struct{ label string }

// NewVote creates a Vote carrying the given label. An empty label yields an
// empty vote.
func NewVote(label string) Vote {
	return Vote{label: label}
}

// VotePtr returns a pointer to a Vote carrying the given label.
func VotePtr(label string) *Vote {
	vote := NewVote(label)
	return &vote
}

// VotePtrFromLabel converts an optional label into an optional Vote. It is the
// nil-safe reverse of OptionalVoteLabel.
func VotePtrFromLabel(label *string) *Vote {
	if label == nil {
		return nil
	}

	return VotePtr(*label)
}

// Label returns the vote label.
func (v Vote) Label() string {
	return v.label
}

// IsEmpty reports whether the vote carries no label.
func (v Vote) IsEmpty() bool {
	return v.label == ""
}

// String returns the vote label.
func (v Vote) String() string {
	return v.Label()
}

// OptionalVoteLabel returns the vote label, or nil when the vote is nil. It
// mirrors OptionalUTCTime.
func OptionalVoteLabel(vote *Vote) *string {
	if vote == nil {
		return nil
	}

	label := vote.Label()
	return &label
}
