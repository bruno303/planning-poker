package entity

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

// Backlog holds the room's ordered stories, the current story index and
// whether backlog mode is enabled.
type Backlog struct {
	roomID       string
	mode         bool
	stories      []Story
	currentIndex int
}

// NewBacklog creates an enabled backlog for the given room.
func NewBacklog(roomID string) *Backlog {
	return &Backlog{roomID: roomID, mode: true}
}

// Mode reports whether backlog mode is enabled.
func (b *Backlog) Mode() bool {
	return b.mode
}

// Stories returns a shallow copy of the backlog stories.
func (b *Backlog) Stories() []Story {
	return slices.Clone(b.stories)
}

// CurrentIndex returns the index of the current story.
func (b *Backlog) CurrentIndex() int {
	return b.currentIndex
}

// Enable turns backlog mode on and seeds a single story from the legacy
// current story when one exists. It reports whether a story was seeded.
func (b *Backlog) Enable(legacyCurrentStory string) bool {
	b.mode = true
	if legacyCurrentStory == "" {
		return false
	}

	b.stories = []Story{{ID: uuid.NewString(), Name: legacyCurrentStory}}
	b.currentIndex = 0
	return true
}

// Disable turns backlog mode off, drops the stories and returns the name of
// the story that was current so it can be preserved as the legacy story.
func (b *Backlog) Disable() string {
	name := b.currentStoryName()
	b.mode = false
	b.stories = nil
	b.currentIndex = 0
	return name
}

// Add appends a new story and enables backlog mode.
func (b *Backlog) Add(name string) Story {
	if !b.mode {
		b.mode = true
	}

	story := Story{ID: uuid.NewString(), Name: name}
	b.stories = append(b.stories, story)
	if len(b.stories) == 1 {
		b.currentIndex = 0
	}
	return story
}

// Remove deletes a story by ID. It reports whether the round must be reset
// because the current story was removed.
func (b *Backlog) Remove(storyID string) (resetRound bool, err error) {
	if storyID == "" {
		return false, fmt.Errorf("story ID cannot be empty")
	}
	index := b.storyIndexByID(storyID)
	if index == -1 {
		return false, fmt.Errorf("story %s not found in room %s", storyID, b.roomID)
	}

	if index == b.currentIndex {
		resetRound = true
		if len(b.stories) == 1 {
			b.currentIndex = 0
			b.stories = nil
			return resetRound, nil
		} else if index == len(b.stories)-1 {
			b.currentIndex--
		}
	} else if index < b.currentIndex {
		b.currentIndex--
	}

	b.stories = append(b.stories[:index], b.stories[index+1:]...)
	return resetRound, nil
}

// Select makes the given pending story the current one.
func (b *Backlog) Select(storyID string) error {
	if storyID == "" {
		return fmt.Errorf("story ID cannot be empty")
	}

	index := b.storyIndexByID(storyID)
	if index == -1 {
		return fmt.Errorf("story %s not found in room %s", storyID, b.roomID)
	}
	if index == b.currentIndex {
		return fmt.Errorf("story %s is already the current story", storyID)
	}
	if b.stories[index].Voted {
		return fmt.Errorf("estimated story %s cannot be selected", storyID)
	}

	b.currentIndex = index
	return nil
}

// Reorder moves a story to targetIndex. The current story remains identified
// by its stable ID.
func (b *Backlog) Reorder(storyID string, targetIndex int) error {
	storyIndex, err := b.validateReorder(storyID, targetIndex)
	if err != nil {
		return err
	}
	if storyIndex == targetIndex {
		return nil
	}

	currentStoryID := b.currentStoryID()
	story := b.stories[storyIndex]
	b.stories = append(b.stories[:storyIndex], b.stories[storyIndex+1:]...)
	b.stories = slices.Insert(b.stories, targetIndex, story)
	b.updateCurrentIndex(storyIndex, targetIndex, currentStoryID)
	return nil
}

func (b *Backlog) validateReorder(storyID string, targetIndex int) (int, error) {
	if storyID == "" {
		return 0, fmt.Errorf("story ID cannot be empty")
	}
	if targetIndex < 0 || targetIndex >= len(b.stories) {
		return 0, fmt.Errorf("target story index %d out of range", targetIndex)
	}

	storyIndex := b.storyIndexByID(storyID)
	if storyIndex == -1 {
		return 0, fmt.Errorf("story %s not found in room %s", storyID, b.roomID)
	}

	return storyIndex, nil
}

func (b *Backlog) currentStoryID() string {
	if b.currentIndex < 0 || b.currentIndex >= len(b.stories) {
		return ""
	}
	return b.stories[b.currentIndex].ID
}

func (b *Backlog) updateCurrentIndex(storyIndex, targetIndex int, currentStoryID string) {
	if currentStoryID != "" {
		b.currentIndex = b.storyIndexByID(currentStoryID)
		return
	}
	if b.currentIndex == storyIndex {
		b.currentIndex = targetIndex
		return
	}
	if storyIndex < b.currentIndex {
		b.currentIndex--
	}
	if targetIndex <= b.currentIndex {
		b.currentIndex++
	}
}

func (b *Backlog) storyIndexByID(storyID string) int {
	for index, story := range b.stories {
		if story.ID == storyID {
			return index
		}
	}
	return -1
}

// Advance moves to the next story and reports whether the index moved.
func (b *Backlog) Advance() bool {
	if b.currentIndex < len(b.stories)-1 {
		b.currentIndex++
		return true
	}
	return false
}

// Back moves to the previous story and reports whether the index moved.
func (b *Backlog) Back() bool {
	if b.currentIndex > 0 {
		b.currentIndex--
		return true
	}
	return false
}

// Current returns the current story when the index is valid.
func (b *Backlog) Current() (Story, bool) {
	if b.currentIndex < 0 || b.currentIndex >= len(b.stories) {
		return Story{}, false
	}
	return b.stories[b.currentIndex], true
}

// SetCurrentName renames the current story when the index is valid.
func (b *Backlog) SetCurrentName(name string) {
	if b.currentIndex < 0 || b.currentIndex >= len(b.stories) {
		return
	}
	b.stories[b.currentIndex].Name = name
}

// EffectiveName returns the current story name while backlog mode is enabled,
// falling back to the legacy current story otherwise.
func (b *Backlog) EffectiveName(fallback string) string {
	if b.mode && b.currentIndex >= 0 && b.currentIndex < len(b.stories) {
		return b.stories[b.currentIndex].Name
	}
	return fallback
}

// ApplyOutcome stamps the round outcome onto the current story.
func (b *Backlog) ApplyOutcome(outcome *RoundOutcome) {
	story, ok := b.Current()
	if !ok || outcome == nil {
		return
	}

	story.Result = outcome.Result
	story.MostAppearingVotes = append([]string(nil), outcome.MostAppearingVotes...)
	story.Voted = true
	b.stories[b.currentIndex] = story
}

func (b *Backlog) currentStoryName() string {
	if b.currentIndex < 0 || b.currentIndex >= len(b.stories) {
		return ""
	}
	return b.stories[b.currentIndex].Name
}

// restore replaces the backlog state without generating story IDs.
func (b *Backlog) restore(mode bool, stories []Story, currentIndex int) {
	b.mode = mode
	b.stories = stories
	b.currentIndex = currentIndex
}
