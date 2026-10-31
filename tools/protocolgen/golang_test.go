package main

import (
	"regexp"
	"strings"
	"testing"

	protocol "planning-poker/internal/protocol"
)

func TestGoName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "room-state", expected: "RoomState"},
		{input: "update-client-id", expected: "UpdateClientID"},
		{input: "commandId", expected: "CommandID"},
		{input: "targetClientId", expected: "TargetClientID"},
		{input: "stale_version", expected: "StaleVersion"},
		{input: "not_found", expected: "NotFound"},
		{input: "invalid_command_id", expected: "InvalidCommandID"},
		{input: "High", expected: "High"},
		{input: "UUIDValue", expected: "UUIDValue"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			if got := goName(test.input); got != test.expected {
				t.Fatalf("goName(%q) = %q, want %q", test.input, got, test.expected)
			}
		})
	}
}

func TestEmitGoTypes(t *testing.T) {
	spec := loadSampleSpec(t)
	output, err := emitGoTypes(spec)
	if err != nil {
		t.Fatalf("emitGoTypes: %v", err)
	}
	generated := string(output)

	if !strings.HasPrefix(generated, generatedHeader+"\n") {
		t.Fatalf("generated file does not start with the header")
	}
	if !strings.Contains(generated, "package protocol") {
		t.Fatal("generated file does not declare package protocol")
	}
	if !strings.Contains(generated, `import "time"`) {
		t.Fatal("generated file does not import time")
	}

	tests := []struct {
		name    string
		pattern string
	}{
		{name: "required nullable scalar is a pointer without omitempty", pattern: "^\tVote\\s+\\*string\\s+\x60json:\"vote\"\x60$"},
		{name: "optional nullable slice keeps the slice without omitempty", pattern: "^\tStories\\s+\\[\\]Story\\s+\x60json:\"stories\"\x60$"},
		{name: "optional float32 hint", pattern: "^\tResult\\s+\\*float32\\s+\x60json:\"result,omitempty\"\x60$"},
		{name: "required uint64 hint", pattern: "^\tRoomVersion\\s+uint64\\s+\x60json:\"roomVersion\"\x60$"},
		{name: "date-time maps to time.Time", pattern: "^\tStartedAt\\s+\\*time.Time\\s+\x60json:\"startedAt,omitempty\"\x60$"},
		{name: "optional integer is a pointer with omitempty", pattern: "^\tCurrentStoryIndex\\s+\\*int\\s+\x60json:\"currentStoryIndex,omitempty\"\x60$"},
		{name: "enum reference is a named type", pattern: "^\tKind\\s+DeckKind\\s+\x60json:\"kind\"\x60$"},
		{name: "initialism for commandId", pattern: "^\tCommandID\\s+string\\s+\x60json:\"commandId\"\x60$"},
		{name: "optional uint64 is a pointer with omitempty", pattern: "^\tRoomVersion\\s+\\*uint64\\s+\x60json:\"roomVersion,omitempty\"\x60$"},
		{name: "initialism for targetClientId", pattern: "^\tTargetClientID\\s+string\\s+\x60json:\"targetClientId\"\x60$"},
		{name: "required integer", pattern: "^\tTargetIndex\\s+int\\s+\x60json:\"targetIndex\"\x60$"},
		{name: "initialism for clientId", pattern: "^\tClientID\\s+string\\s+\x60json:\"clientId\"\x60$"},
		{name: "const becomes a plain type field", pattern: "^\tType\\s+string\\s+\x60json:\"type\"\x60$"},
		{name: "deck kind type", pattern: "^\tDeckKindNumeric\\s+DeckKind = \"numeric\"$"},
		{name: "consensus enum type", pattern: "^\tConsensusLevelUnavailable\\s+ConsensusLevel = \"Unavailable\"$"},
		{name: "snake case enum constant", pattern: "^\tCommandErrorCodeStaleVersion\\s+CommandErrorCode = \"stale_version\"$"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pattern := regexp.MustCompile("(?m)" + test.pattern)
			if !pattern.MatchString(generated) {
				t.Fatalf("generated Go types do not match %s", test.pattern)
			}
		})
	}
}

func TestEmitGoMessages(t *testing.T) {
	spec := loadSampleSpec(t)
	output, err := emitGoMessages(spec)
	if err != nil {
		t.Fatalf("emitGoMessages: %v", err)
	}
	generated := string(output)

	if !strings.HasPrefix(generated, generatedHeader+"\n") {
		t.Fatalf("generated file does not start with the header")
	}

	tests := []struct {
		name    string
		pattern string
	}{
		{name: "room state constant", pattern: "^\tMessageTypeRoomState\\s+= \"room-state\"$"},
		{name: "client id constant", pattern: "^\tMessageTypeUpdateClientID\\s+= \"update-client-id\"$"},
		{name: "client commands list", pattern: "^\tClientCommandTypes = \\[\\]string\\{$"},
		{name: "server events list", pattern: "^\tServerEventTypes = \\[\\]string\\{$"},
		{name: "payload constructor entry", pattern: "^\t\treturn &ReorderStoryPayload\\{\\}, true$"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pattern := regexp.MustCompile("(?m)" + test.pattern)
			if !pattern.MatchString(generated) {
				t.Fatalf("generated Go messages do not match %s", test.pattern)
			}
		})
	}

	if got := strings.Count(generated, "\t\t\""); got != 21 {
		t.Fatalf("type lists contain %d entries, want 21", got)
	}
	for _, function := range []string{
		"func IsClientCommandType(messageType string) bool",
		"func IsServerEventType(messageType string) bool",
		"func NewClientCommandPayload(messageType string) (any, bool)",
	} {
		if !strings.Contains(generated, function) {
			t.Fatalf("generated Go messages are missing %q", function)
		}
	}
}

func TestGoGenerationIsDeterministic(t *testing.T) {
	spec := loadSampleSpec(t)

	firstTypes, err := emitGoTypes(spec)
	if err != nil {
		t.Fatalf("emitGoTypes: %v", err)
	}
	secondTypes, err := emitGoTypes(spec)
	if err != nil {
		t.Fatalf("emitGoTypes: %v", err)
	}
	if string(firstTypes) != string(secondTypes) {
		t.Fatal("emitGoTypes is not deterministic")
	}

	firstMessages, err := emitGoMessages(spec)
	if err != nil {
		t.Fatalf("emitGoMessages: %v", err)
	}
	secondMessages, err := emitGoMessages(spec)
	if err != nil {
		t.Fatalf("emitGoMessages: %v", err)
	}
	if string(firstMessages) != string(secondMessages) {
		t.Fatal("emitGoMessages is not deterministic")
	}
}

func TestGeneratedGoProtocolBehaviour(t *testing.T) {
	payload, ok := protocol.NewClientCommandPayload(protocol.MessageTypeReorderStory)
	if !ok {
		t.Fatal("NewClientCommandPayload rejected reorder-story")
	}
	if _, isReorder := payload.(*protocol.ReorderStoryPayload); !isReorder {
		t.Fatalf("NewClientCommandPayload returned %T, want *protocol.ReorderStoryPayload", payload)
	}

	if _, ok := protocol.NewClientCommandPayload("room-state"); ok {
		t.Fatal("NewClientCommandPayload accepted a server event")
	}
	if !protocol.IsClientCommandType("vote") {
		t.Fatal("IsClientCommandType rejected vote")
	}
	if protocol.IsClientCommandType("room-state") {
		t.Fatal("IsClientCommandType accepted room-state")
	}
	if !protocol.IsServerEventType("room-state") {
		t.Fatal("IsServerEventType rejected room-state")
	}
	if protocol.IsServerEventType("vote") {
		t.Fatal("IsServerEventType accepted vote")
	}
	if len(protocol.ClientCommandTypes) != 15 || len(protocol.ServerEventTypes) != 6 {
		t.Fatalf(
			"generated lists = %d/%d, want 15/6",
			len(protocol.ClientCommandTypes),
			len(protocol.ServerEventTypes),
		)
	}
}
