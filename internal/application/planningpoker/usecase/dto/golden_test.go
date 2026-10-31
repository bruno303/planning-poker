package dto

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const roomStateGoldenPath = "testdata/room-state.golden.json"

var updateGolden = flag.Bool("update", false, "rewrite the room-state golden fixture")

func TestRoomStateGolden(t *testing.T) {
	got, err := json.Marshal(NewRoomStateCommand(goldenRoom()))
	require.NoError(t, err)

	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(roomStateGoldenPath), 0o755))
		require.NoError(t, os.WriteFile(roomStateGoldenPath, append(got, '\n'), 0o644))
		t.Logf("updated golden fixture at %s", roomStateGoldenPath)
		return
	}

	want, err := os.ReadFile(roomStateGoldenPath)
	require.NoError(t, err)

	assert.JSONEq(t, string(want), string(got))
}

func goldenRoom() *entity.Room {
	startedAt := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC)
	votedAt := time.Date(2026, time.September, 3, 12, 1, 0, 0, time.UTC)
	vote := "5"

	clients := clientcollection.New()
	clients.Add(&entity.Client{
		ID:          "client-alice",
		Name:        "Alice",
		CurrentVote: &vote,
		HasVoted:    true,
		VotedAt:     &votedAt,
		IsOwner:     true,
	})
	clients.Add(&entity.Client{
		ID:          "client-bob",
		Name:        "Bob",
		IsSpectator: true,
	})

	room := entity.NewRoomWithIDAndStartedAt("room-golden", clients, startedAt)
	room.CurrentStory = "Story 1"
	room.Reveal = true
	room.Result = lo.ToPtr(float32(5))
	room.MostAppearingVotes = []string{"5", "3"}
	room.Consensus = "Medium"
	room.LowestVote = lo.ToPtr("3")
	room.HighestVote = lo.ToPtr("8")
	room.VoteRange = lo.ToPtr(5)
	room.VoteSpread = lo.ToPtr(2)
	room.SpecialVoteCount = 1
	room.BacklogMode = true
	room.RoomVersion = 4
	room.Stories = []entity.Story{
		{
			ID:                 "story-1",
			Name:               "Story 1",
			Result:             lo.ToPtr(float32(5)),
			MostAppearingVotes: []string{"5", "3"},
			Voted:              true,
		},
	}

	return room
}
