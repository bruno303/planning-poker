package entity

type Story struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Result          *float32 `json:"result,omitempty"`
	MostCommonVotes []string `json:"mostCommonVotes"`
	Voted           bool     `json:"voted"`
}
