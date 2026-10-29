package entity

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

func (r *Room) ToggleBacklogMode(ctx context.Context, clientID ClientID) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can toggle backlog mode")
	}

	if !r.BacklogMode {
		r.BacklogMode = true
		if r.CurrentStory != "" {
			r.Stories = []Story{{ID: StoryID(uuid.NewString()), Name: r.CurrentStory}}
			r.CurrentStoryIndex = 0
		}
	} else {
		r.BacklogMode = false
		if name := r.getCurrentStoryName(); name != "" {
			r.CurrentStory = name
		}
		r.Stories = nil
		r.CurrentStoryIndex = 0
	}
	return nil
}

func (r *Room) AddStory(ctx context.Context, clientID ClientID, name string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can add a story")
	}

	if !r.BacklogMode {
		r.BacklogMode = true
	}

	r.Stories = append(r.Stories, Story{ID: StoryID(uuid.NewString()), Name: name})
	if len(r.Stories) == 1 {
		r.CurrentStoryIndex = 0
	}
	return nil
}

func (r *Room) RemoveStory(ctx context.Context, clientID ClientID, storyID StoryID) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can remove a story")
	}
	if storyID == "" {
		return fmt.Errorf("story ID cannot be empty")
	}
	index := r.storyIndexByID(storyID)
	if index == -1 {
		return fmt.Errorf("story %s not found in room %s", storyID, r.ID)
	}

	if index == r.CurrentStoryIndex {
		if len(r.Stories) == 1 {
			r.CurrentStoryIndex = 0
			r.Stories = nil
			r.reveal(false)
			r.Clients.ForEach(func(c *Client) {
				r.applyVote(c, nil)
			})
			return nil
		} else if index == len(r.Stories)-1 {
			r.CurrentStoryIndex--
		}
		r.reveal(false)
		r.Clients.ForEach(func(c *Client) {
			r.applyVote(c, nil)
		})
	} else if index < r.CurrentStoryIndex {
		r.CurrentStoryIndex--
	}

	r.Stories = append(r.Stories[:index], r.Stories[index+1:]...)
	return nil
}

// SelectStory selects a pending story and starts a fresh voting round.
func (r *Room) SelectStory(ctx context.Context, clientID ClientID, storyID StoryID) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can select a story")
	}
	if storyID == "" {
		return fmt.Errorf("story ID cannot be empty")
	}

	index := r.storyIndexByID(storyID)
	if index == -1 {
		return fmt.Errorf("story %s not found in room %s", storyID, r.ID)
	}
	if index == r.CurrentStoryIndex {
		return fmt.Errorf("story %s is already the current story", storyID)
	}
	if r.Stories[index].Voted {
		return fmt.Errorf("estimated story %s cannot be selected", storyID)
	}

	r.CurrentStoryIndex = index
	r.reveal(false)
	r.Clients.ForEach(func(c *Client) {
		r.applyVote(c, nil)
	})

	r.recordEvent(StorySelected{
		RoomID:     r.ID,
		StoryID:    storyID,
		OccurredAt: time.Now().UTC(),
	})

	return nil
}

// ReorderStory moves a story to targetIndex. The current story remains
// identified by its stable ID.
func (r *Room) ReorderStory(ctx context.Context, clientID ClientID, storyID StoryID, targetIndex int) error {
	storyIndex, err := r.validateReorderStory(clientID, storyID, targetIndex)
	if err != nil {
		return err
	}
	if storyIndex == targetIndex {
		return nil
	}

	currentStoryID := r.currentStoryID()
	story := r.Stories[storyIndex]
	r.Stories = append(r.Stories[:storyIndex], r.Stories[storyIndex+1:]...)
	r.Stories = slices.Insert(r.Stories, targetIndex, story)
	r.updateCurrentStoryIndex(storyIndex, targetIndex, currentStoryID)
	return nil
}

func (r *Room) validateReorderStory(clientID ClientID, storyID StoryID, targetIndex int) (int, error) {
	client, ok := r.FindClient(clientID)
	if !ok {
		return 0, fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return 0, fmt.Errorf("only the room owner can reorder stories")
	}
	if storyID == "" {
		return 0, fmt.Errorf("story ID cannot be empty")
	}
	if targetIndex < 0 || targetIndex >= len(r.Stories) {
		return 0, fmt.Errorf("target story index %d out of range", targetIndex)
	}

	storyIndex := r.storyIndexByID(storyID)
	if storyIndex == -1 {
		return 0, fmt.Errorf("story %s not found in room %s", storyID, r.ID)
	}

	return storyIndex, nil
}

func (r *Room) currentStoryID() StoryID {
	if r.CurrentStoryIndex < 0 || r.CurrentStoryIndex >= len(r.Stories) {
		return ""
	}
	return r.Stories[r.CurrentStoryIndex].ID
}

func (r *Room) updateCurrentStoryIndex(storyIndex, targetIndex int, currentStoryID StoryID) {
	if currentStoryID != "" {
		r.CurrentStoryIndex = r.storyIndexByID(currentStoryID)
		return
	}
	if r.CurrentStoryIndex == storyIndex {
		r.CurrentStoryIndex = targetIndex
		return
	}
	if storyIndex < r.CurrentStoryIndex {
		r.CurrentStoryIndex--
	}
	if targetIndex <= r.CurrentStoryIndex {
		r.CurrentStoryIndex++
	}
}

func (r *Room) storyIndexByID(storyID StoryID) int {
	for index, story := range r.Stories {
		if story.ID == storyID {
			return index
		}
	}
	return -1
}

func (r *Room) AdvanceToNextStory(ctx context.Context, clientID ClientID) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can advance to the next story")
	}

	if r.CurrentStoryIndex < len(r.Stories)-1 {
		r.CurrentStoryIndex++
		r.reveal(false)
		r.Clients.ForEach(func(c *Client) {
			r.applyVote(c, nil)
		})
	}
	return nil
}

func (r *Room) PrevStory(ctx context.Context, clientID ClientID) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can go to the previous story")
	}

	if r.CurrentStoryIndex > 0 {
		r.CurrentStoryIndex--
		r.reveal(false)
		r.Clients.ForEach(func(c *Client) {
			r.applyVote(c, nil)
		})
	}
	return nil
}

func (r *Room) EffectiveCurrentStory() string {
	if r.BacklogMode && len(r.Stories) > 0 && r.CurrentStoryIndex < len(r.Stories) {
		return r.Stories[r.CurrentStoryIndex].Name
	}
	return r.CurrentStory
}

func (r *Room) getCurrentStoryName() string {
	if len(r.Stories) > 0 && r.CurrentStoryIndex < len(r.Stories) {
		return r.Stories[r.CurrentStoryIndex].Name
	}
	return ""
}

func (r *Room) SetCurrentStory(ctx context.Context, clientID ClientID, story string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can set the current story")
	}

	if r.BacklogMode && r.CurrentStoryIndex >= 0 && r.CurrentStoryIndex < len(r.Stories) {
		r.Stories[r.CurrentStoryIndex].Name = story
	} else {
		r.CurrentStory = story
	}
	return nil
}
