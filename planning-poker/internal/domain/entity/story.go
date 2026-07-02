package entity

type Story struct {
	Name               string   `json:"name"`
	Result             *float32 `json:"result,omitempty"`
	MostAppearingVotes []int    `json:"mostAppearingVotes"`
	Voted              bool     `json:"voted"`
}
