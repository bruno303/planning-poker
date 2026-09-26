package entity

type Story struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Result             *float64 `json:"result,omitempty"`
	MostAppearingVotes []string `json:"mostAppearingVotes"`
	Voted              bool     `json:"voted"`
}
