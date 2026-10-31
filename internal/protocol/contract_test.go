package protocol_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"planning-poker/internal/protocol"
)

const schemaBaseURL = "https://planning-poker.local/protocol/schemas/"

var serverEventSchemaDefs = map[string]string{
	protocol.MessageTypeCommandAck:     "CommandAck",
	protocol.MessageTypeCommandError:   "CommandError",
	protocol.MessageTypeKicked:         "Kicked",
	protocol.MessageTypeRoomState:      "RoomState",
	protocol.MessageTypeStaleCommand:   "StaleCommand",
	protocol.MessageTypeUpdateClientID: "UpdateClientID",
}

// invalidClientCommandTypes maps invalid client fixtures to the command they
// target. Valid fixtures are named after their command type.
var invalidClientCommandTypes = map[string]string{
	"vote-number.json":                    protocol.MessageTypeVote,
	"reorder-target-index-string.json":    protocol.MessageTypeReorderStory,
	"toggle-spectator-target-number.json": protocol.MessageTypeToggleSpectator,
}

func TestServerFixturesMatchSchemas(t *testing.T) {
	compiler := compileProtocolSchemas(t)
	serverDir := filepath.Join(protocolDir(t), "fixtures", "server")

	type fixtureCase struct {
		path    string
		wantErr bool
	}
	cases := make([]fixtureCase, 0, 16)
	for _, name := range jsonFiles(t, serverDir) {
		cases = append(cases, fixtureCase{path: filepath.Join(serverDir, name)})
	}
	invalidDir := filepath.Join(serverDir, "invalid")
	for _, name := range jsonFiles(t, invalidDir) {
		cases = append(cases, fixtureCase{path: filepath.Join(invalidDir, name), wantErr: true})
	}
	if len(cases) == 0 {
		t.Fatal("no server fixtures found")
	}

	for _, tc := range cases {
		name := filepath.Base(tc.path)
		t.Run(name, func(t *testing.T) {
			eventType, ok := serverEventTypeForFixture(name)
			if !ok {
				t.Fatalf("cannot determine server event type for fixture %q", name)
			}
			schema := compileSchema(t, compiler, "events.schema.json", serverEventSchemaDefs[eventType])

			err := schema.Validate(readJSONDocument(t, tc.path))
			if tc.wantErr && err == nil {
				t.Fatalf("schema accepted invalid server fixture %q", name)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("schema rejected valid server fixture %q: %v", name, err)
			}
		})
	}
}

func TestClientFixturesMatchSchemas(t *testing.T) {
	compiler := compileProtocolSchemas(t)
	clientDir := filepath.Join(protocolDir(t), "fixtures", "client")

	valid := jsonFiles(t, clientDir)
	invalid := jsonFiles(t, filepath.Join(clientDir, "invalid"))
	if len(valid) == 0 {
		t.Fatal("no valid client fixtures found")
	}
	if len(invalid) == 0 {
		t.Fatal("no invalid client fixtures found")
	}

	for _, name := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			commandType := strings.TrimSuffix(name, ".json")
			schema := compileSchema(t, compiler, "commands.schema.json", schemaDefForCommandOrFail(t, commandType))

			if err := schema.Validate(readJSONDocument(t, filepath.Join(clientDir, name))); err != nil {
				t.Fatalf("schema rejected valid client fixture %q: %v", name, err)
			}
		})
	}

	for _, name := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			commandType, ok := invalidClientCommandTypes[name]
			if !ok {
				t.Fatalf("invalid client fixture %q has no registered command type", name)
			}
			schema := compileSchema(t, compiler, "commands.schema.json", schemaDefForCommandOrFail(t, commandType))

			if err := schema.Validate(readJSONDocument(t, filepath.Join(clientDir, "invalid", name))); err == nil {
				t.Fatalf("schema accepted invalid client fixture %q", name)
			}
		})
	}
}

// TestClientPayloadDecoding decodes the payload of every valid client fixture
// into the generated payload struct and asserts the decoded values. Every
// invalid fixture must fail decoding.
//
// Note that Go's json.Unmarshal does not reject missing required fields: absent
// properties simply keep their zero value. Required-field enforcement for
// client frames comes from the schema validation in
// TestClientFixturesMatchSchemas and from use-case validation at runtime.
func TestClientPayloadDecoding(t *testing.T) {
	clientDir := filepath.Join(protocolDir(t), "fixtures", "client")
	expected := expectedClientPayloads()

	seenCommandIDs := make(map[string]string)
	for _, name := range jsonFiles(t, clientDir) {
		t.Run("valid/"+name, func(t *testing.T) {
			fixture := readClientFixture(t, filepath.Join(clientDir, name))
			commandType := strings.TrimSuffix(name, ".json")

			if fixture.Type != commandType {
				t.Fatalf("fixture type %q does not match file name command %q", fixture.Type, commandType)
			}
			if fixture.CommandID == nil || *fixture.CommandID == "" {
				t.Fatalf("fixture %q has no commandId", name)
			}
			if other, ok := seenCommandIDs[*fixture.CommandID]; ok {
				t.Fatalf("fixture %q reuses the commandId of %q", name, other)
			}
			seenCommandIDs[*fixture.CommandID] = name

			payload, ok := protocol.NewClientCommandPayload(fixture.Type)
			if !ok {
				t.Fatalf("no generated payload for command type %q", fixture.Type)
			}
			if err := json.Unmarshal(fixture.Payload, payload); err != nil {
				t.Fatalf("decode payload of fixture %q: %v", name, err)
			}

			want, ok := expected[fixture.Type]
			if !ok {
				t.Fatalf("no expected payload registered for command type %q", fixture.Type)
			}
			if !reflect.DeepEqual(payload, want) {
				t.Fatalf("decoded payload of fixture %q = %#v, want %#v", name, payload, want)
			}
		})
	}

	invalidDir := filepath.Join(clientDir, "invalid")
	for _, name := range jsonFiles(t, invalidDir) {
		t.Run("invalid/"+name, func(t *testing.T) {
			fixture := readClientFixture(t, filepath.Join(invalidDir, name))

			payload, ok := protocol.NewClientCommandPayload(fixture.Type)
			if !ok {
				t.Fatalf("no generated payload for command type %q", fixture.Type)
			}
			if err := json.Unmarshal(fixture.Payload, payload); err == nil {
				t.Fatalf("payload of invalid fixture %q decoded without error", name)
			}
		})
	}
}

type clientFixture struct {
	Type      string          `json:"type"`
	CommandID *string         `json:"commandId"`
	Payload   json.RawMessage `json:"payload"`
}

func readClientFixture(t *testing.T, path string) clientFixture {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read client fixture %q: %v", path, err)
	}

	var fixture clientFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode client fixture %q: %v", path, err)
	}
	return fixture
}

func expectedClientPayloads() map[string]any {
	version := uint64(12)
	expectedVersion := &version

	return map[string]any{
		protocol.MessageTypeAddStory:          &protocol.AddStoryPayload{Story: "Estimate the login flow"},
		protocol.MessageTypeAdvanceStory:      &protocol.AdvanceStoryPayload{ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypePrevStory:         &protocol.PrevStoryPayload{ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypeRemoveStory:       &protocol.RemoveStoryPayload{StoryID: "story-2", ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypeReorderStory:      &protocol.ReorderStoryPayload{StoryID: "story-2", TargetIndex: 2, ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypeReset:             &protocol.ResetPayload{ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypeRevealVotes:       &protocol.RevealVotesPayload{ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypeSelectStory:       &protocol.SelectStoryPayload{StoryID: "story-2", ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypeToggleBacklogMode: &protocol.ToggleBacklogModePayload{ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypeToggleOwner:       &protocol.ToggleOwnerPayload{TargetClientID: "client-ada", ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypeToggleSpectator:   &protocol.ToggleSpectatorPayload{TargetClientID: "client-grace", ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypeUpdateName:        &protocol.UpdateNamePayload{Username: "Ada Lovelace"},
		protocol.MessageTypeUpdateStory:       &protocol.UpdateStoryPayload{Story: "Estimate the checkout flow", ExpectedRoomVersion: expectedVersion},
		protocol.MessageTypeVote:              &protocol.VotePayload{Vote: "5"},
		protocol.MessageTypeVoteAgain:         &protocol.VoteAgainPayload{ExpectedRoomVersion: expectedVersion},
	}
}

func serverEventTypeForFixture(name string) (string, bool) {
	base := strings.TrimSuffix(name, ".json")
	for eventType := range serverEventSchemaDefs {
		if base == eventType || strings.HasPrefix(base, eventType+".") || strings.HasPrefix(base, eventType+"-") {
			return eventType, true
		}
	}
	return "", false
}

func schemaDefForCommand(commandType string) (string, bool) {
	parts := strings.Split(commandType, "-")
	for i, part := range parts {
		if part == "" {
			return "", false
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "") + "Command", true
}

func schemaDefForCommandOrFail(t *testing.T, commandType string) string {
	t.Helper()

	def, ok := schemaDefForCommand(commandType)
	if !ok {
		t.Fatalf("cannot derive schema definition for command type %q", commandType)
	}
	return def
}

func compileProtocolSchemas(t *testing.T) *jsonschema.Compiler {
	t.Helper()

	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()

	schemaDir := filepath.Join(protocolDir(t), "schemas")
	for _, name := range jsonFiles(t, schemaDir) {
		if !strings.HasSuffix(name, ".schema.json") {
			continue
		}
		f, err := os.Open(filepath.Join(schemaDir, name))
		if err != nil {
			t.Fatalf("open schema %q: %v", name, err)
		}
		doc, err := jsonschema.UnmarshalJSON(f)
		closeErr := f.Close()
		if err != nil {
			t.Fatalf("decode schema %q: %v", name, err)
		}
		if closeErr != nil {
			t.Fatalf("close schema %q: %v", name, closeErr)
		}
		if err := compiler.AddResource(schemaBaseURL+name, doc); err != nil {
			t.Fatalf("register schema %q: %v", name, err)
		}
	}

	return compiler
}

func compileSchema(t *testing.T, compiler *jsonschema.Compiler, file, def string) *jsonschema.Schema {
	t.Helper()

	loc := schemaBaseURL + file + "#/$defs/" + def
	schema, err := compiler.Compile(loc)
	if err != nil {
		t.Fatalf("compile schema %q: %v", loc, err)
	}
	return schema
}

func readJSONDocument(t *testing.T, path string) any {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture %q: %v", path, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatalf("close fixture %q: %v", path, err)
		}
	}()

	doc, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		t.Fatalf("decode fixture %q: %v", path, err)
	}
	return doc
}

func jsonFiles(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixture directory %q: %v", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}

func protocolDir(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve the contract test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "protocol"))
}
