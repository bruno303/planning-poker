package dto

import (
	"slices"
	"strings"

	"planning-poker/internal/domain/entity"
	"planning-poker/internal/protocol"

	"github.com/samber/lo"
)

func NewRoomStateCommand(room *entity.Room) protocol.RoomState {
	deck := room.Deck()
	return protocol.RoomState{
		Type:      protocol.MessageTypeRoomState,
		StartedAt: entity.OptionalUTCTime(room.StartedAt()),
		Deck: protocol.DeckDescriptor{
			ID:    deck.ID,
			Name:  deck.Name,
			Kind:  protocol.DeckKind(deck.Kind),
			Cards: deck.Cards,
		},
		CurrentStory:       room.EffectiveCurrentStory(),
		Reveal:             room.Reveal,
		Participants:       MapToParticipants(room.Clients.Values()),
		Result:             room.Result,
		MostAppearingVotes: room.MostAppearingVotes,
		Consensus:          optionalConsensus(room.Consensus),
		LowestVote:         room.LowestVote,
		HighestVote:        room.HighestVote,
		VoteRange:          room.VoteRange,
		VoteSpread:         room.VoteSpread,
		SpecialVoteCount:   optionalInt(room.SpecialVoteCount),
		BacklogMode:        lo.ToPtr(room.BacklogMode),
		Stories:            mapStories(room.Stories),
		CurrentStoryIndex:  lo.ToPtr(room.CurrentStoryIndex),
		RoomVersion:        room.RoomVersion,
	}
}

func NewUpdateClientIDCommand(clientID string) protocol.UpdateClientID {
	return protocol.UpdateClientID{
		Type:     protocol.MessageTypeUpdateClientID,
		ClientID: clientID,
	}
}

func NewKickNotification() protocol.Kicked {
	return protocol.Kicked{
		Type: protocol.MessageTypeKicked,
	}
}

func optionalConsensus(consensus string) *protocol.ConsensusLevel {
	if consensus == "" {
		return nil
	}
	return lo.ToPtr(protocol.ConsensusLevel(consensus))
}

func optionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return lo.ToPtr(value)
}

func mapStories(stories []entity.Story) []protocol.Story {
	return lo.Map(stories, func(s entity.Story, _ int) protocol.Story {
		return protocol.Story{
			ID:                 s.ID,
			Name:               s.Name,
			Result:             s.Result,
			MostAppearingVotes: s.MostAppearingVotes,
			Voted:              s.Voted,
		}
	})
}

func MapToParticipants(clients []*entity.Client) []protocol.Participant {
	slices.SortFunc(clients, func(a, b *entity.Client) int {
		return strings.Compare(a.Name, b.Name)
	})

	return lo.Map(
		clients,
		func(client *entity.Client, _ int) protocol.Participant {
			return protocol.Participant{
				ID:          client.ID,
				Name:        client.Name,
				Vote:        client.CurrentVote,
				HasVoted:    client.HasVoted,
				VotedAt:     entity.OptionalUTCTimePtr(client.VotedAt),
				IsSpectator: client.IsSpectator,
				IsOwner:     client.IsOwner,
			}
		},
	)
}
