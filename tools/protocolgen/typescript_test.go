package main

import (
	"regexp"
	"strings"
	"testing"
)

func TestEmitTypeScript(t *testing.T) {
	spec := loadSampleSpec(t)
	output, err := emitTypeScript(spec)
	if err != nil {
		t.Fatalf("emitTypeScript: %v", err)
	}
	generated := string(output)

	if !strings.HasPrefix(generated, generatedHeader+"\n") {
		t.Fatalf("generated file does not start with the header")
	}
	if !strings.Contains(generated, "import { isInteger, isRecord, isRFC3339Timestamp, isUUID } from './protocol.runtime';") {
		t.Fatal("generated file does not import every runtime helper it uses")
	}
	if regexp.MustCompile(`\bany\b`).MatchString(generated) {
		t.Fatal("generated TypeScript must not use any")
	}

	tests := []struct {
		name    string
		pattern string
	}{
		{name: "optional nullable slice", pattern: `^  stories\?: Story\[\] \| null;$`},
		{name: "required nullable slice", pattern: `^  mostAppearingVotes: string\[\] \| null;$`},
		{name: "required nullable scalar", pattern: `^  vote: string \| null;$`},
		{name: "optional uv", pattern: `^  expectedRoomVersion\?: number;$`},
		{name: "const type", pattern: `^  type: 'room-state';$`},
		{name: "optional frame command id", pattern: `^  commandId\?: string;$`},
		{name: "optional check uses the in operator", pattern: `^    \(!\('roomVersion' in value\) \|\| isInteger\(value\.roomVersion\)\) &&$`},
		{name: "optional stories accepts absent or null", pattern: `^    \(!\('stories' in value\) \|\| \(value\.stories === null \|\| \(Array\.isArray\(value\.stories\) && value\.stories\.every\(\(item\) => isStory\(item\)\)\)\)\) &&$`},
		{name: "required nullable scalar guard", pattern: `^    \(value\.vote === null \|\| \(typeof value\.vote === 'string'\)\) &&$`},
		{name: "date-time uses the runtime guard", pattern: `^    \(!\('startedAt' in value\) \|\| isRFC3339Timestamp\(value\.startedAt\)\) &&$`},
		{name: "uuid uses the runtime guard", pattern: `^    isUUID\(value\.commandId\) &&$`},
		{name: "integer uses the runtime guard", pattern: `^    isInteger\(value\.targetIndex\)$`},
		{name: "enum reference uses its guard", pattern: `^    isDeckKind\(value\.kind\) &&$`},
		{name: "enum alias", pattern: `^export type DeckKind = 'numeric' \| 'categorical';$`},
		{name: "consensus alias", pattern: `^export type ConsensusLevel = 'High' \| 'Medium' \| 'Low' \| 'Unavailable';$`},
		{name: "client command union", pattern: `^export type ClientCommandType =$`},
		{name: "server event union", pattern: `^export type ServerEventType =$`},
		{name: "guarded command union", pattern: `^export type GuardedCommandType =$`},
		{name: "command frame union", pattern: `^export type ClientCommand =$`},
		{name: "event frame union", pattern: `^export type ServerEvent =$`},
		{name: "guarded list", pattern: `^export const GUARDED_COMMAND_TYPES: readonly GuardedCommandType\[\] = \[$`},
		{name: "server event list", pattern: `^export const SERVER_EVENT_TYPES: readonly ServerEventType\[\] = \[$`},
		{name: "guarded predicate", pattern: `^export function isGuardedCommand\(type: string\): type is GuardedCommandType \{$`},
		{name: "server event dispatcher", pattern: `^export function isServerEvent\(value: unknown\): value is ServerEvent \{$`},
		{name: "client command dispatcher", pattern: `^export function isClientCommand\(value: unknown\): value is ClientCommand \{$`},
		{name: "dispatcher case", pattern: `^    case 'room-state':\n      return isRoomState\(value\);$`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pattern := regexp.MustCompile("(?m)" + test.pattern)
			if !pattern.MatchString(generated) {
				t.Fatalf("generated TypeScript does not match %s", test.pattern)
			}
		})
	}

	for _, guard := range []string{
		"isRoomState",
		"isStory",
		"isParticipant",
		"isDeckDescriptor",
		"isDeckKind",
		"isConsensusLevel",
		"isCommandErrorCode",
		"isUpdateClientID",
		"isKicked",
		"isStaleCommand",
		"isCommandAck",
		"isCommandError",
		"isVotePayload",
		"isUpdateNamePayload",
		"isResetPayload",
		"isRevealVotesPayload",
		"isToggleSpectatorPayload",
		"isToggleOwnerPayload",
		"isUpdateStoryPayload",
		"isVoteAgainPayload",
		"isToggleBacklogModePayload",
		"isAddStoryPayload",
		"isRemoveStoryPayload",
		"isAdvanceStoryPayload",
		"isPrevStoryPayload",
		"isSelectStoryPayload",
		"isReorderStoryPayload",
		"isVoteCommand",
		"isUpdateNameCommand",
		"isResetCommand",
		"isRevealVotesCommand",
		"isToggleSpectatorCommand",
		"isToggleOwnerCommand",
		"isUpdateStoryCommand",
		"isVoteAgainCommand",
		"isToggleBacklogModeCommand",
		"isAddStoryCommand",
		"isRemoveStoryCommand",
		"isAdvanceStoryCommand",
		"isPrevStoryCommand",
		"isSelectStoryCommand",
		"isReorderStoryCommand",
	} {
		if !strings.Contains(generated, "export function "+guard+"(") {
			t.Errorf("generated TypeScript is missing %s", guard)
		}
	}
}

func TestEmitTypeScriptUnions(t *testing.T) {
	spec := loadSampleSpec(t)
	output, err := emitTypeScript(spec)
	if err != nil {
		t.Fatalf("emitTypeScript: %v", err)
	}
	generated := string(output)

	if got := strings.Count(generated, "  | '"); got != 33 {
		t.Fatalf("wire type unions contain %d entries, want 33 (15 commands + 6 events + 12 guarded)", got)
	}

	guarded := between(t, generated, "export type GuardedCommandType =", ";")
	for _, wireType := range []string{"vote", "update-name", "add-story"} {
		if strings.Contains(guarded, "'"+wireType+"'") {
			t.Errorf("guarded union unexpectedly contains %q", wireType)
		}
	}
	for _, wireType := range []string{"vote-again", "reset", "reorder-story"} {
		if !strings.Contains(guarded, "'"+wireType+"'") {
			t.Errorf("guarded union is missing %q", wireType)
		}
	}

	events := between(t, generated, "export type ServerEventType =", ";")
	if got := strings.Count(events, "  | '"); got != 6 {
		t.Fatalf("server event union has %d entries, want 6", got)
	}

	commands := between(t, generated, "export type ClientCommandType =", ";")
	if got := strings.Count(commands, "  | '"); got != 15 {
		t.Fatalf("client command union has %d entries, want 15", got)
	}
}

func TestTypeScriptGenerationIsDeterministic(t *testing.T) {
	spec := loadSampleSpec(t)

	first, err := emitTypeScript(spec)
	if err != nil {
		t.Fatalf("emitTypeScript: %v", err)
	}
	second, err := emitTypeScript(spec)
	if err != nil {
		t.Fatalf("emitTypeScript: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("emitTypeScript is not deterministic")
	}
}

func between(t *testing.T, value, start, end string) string {
	t.Helper()
	startIndex := strings.Index(value, start)
	if startIndex < 0 {
		t.Fatalf("%q not found", start)
	}
	rest := value[startIndex+len(start):]
	endIndex := strings.Index(rest, end)
	if endIndex < 0 {
		t.Fatalf("%q not found after %q", end, start)
	}
	return rest[:endIndex]
}
