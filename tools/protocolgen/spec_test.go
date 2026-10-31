package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	root, err := findRepoRoot(workingDirectory)
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	return root
}

func loadSampleSpec(t *testing.T) *Spec {
	t.Helper()
	spec, err := loadSpec(repoRoot(t))
	if err != nil {
		t.Fatalf("loadSpec: %v", err)
	}
	return spec
}

func messageByType(t *testing.T, spec *Spec, wireType string) Message {
	t.Helper()
	for _, message := range spec.Messages {
		if message.WireType == wireType {
			return message
		}
	}
	t.Fatalf("message %q not found", wireType)
	return Message{}
}

func definitionByName(t *testing.T, spec *Spec, name string) *Definition {
	t.Helper()
	definition, ok := spec.Definition(name)
	if !ok {
		t.Fatalf("definition %q not found", name)
	}
	return definition
}

func copyTree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy %s to %s: %v", source, destination, err)
	}
}

func tempRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n\ngo 1.25.4\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	copyTree(t, filepath.Join(repoRoot(t), "protocol"), filepath.Join(root, "protocol"))
	return root
}

func mutateFile(t *testing.T, path, oldValue, newValue string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	updated := strings.Replace(string(data), oldValue, newValue, 1)
	if updated == string(data) {
		t.Fatalf("%s does not contain %q", path, oldValue)
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoadSpec(t *testing.T) {
	spec := loadSampleSpec(t)

	if len(spec.Messages) != 21 {
		t.Fatalf("messages = %d, want 21", len(spec.Messages))
	}
	if got := len(spec.CommandMessages()); got != 15 {
		t.Fatalf("commands = %d, want 15", got)
	}
	if got := len(spec.EventMessages()); got != 6 {
		t.Fatalf("events = %d, want 6", got)
	}
	if got := len(spec.Definitions); got != 42 {
		t.Fatalf("definitions = %d, want 42", got)
	}
	for index := 1; index < len(spec.Messages); index++ {
		if spec.Messages[index-1].WireType >= spec.Messages[index].WireType {
			t.Fatalf("messages are not sorted: %q before %q", spec.Messages[index-1].WireType, spec.Messages[index].WireType)
		}
	}

	roomState := messageByType(t, spec, "room-state")
	if roomState.Direction != DirectionReceive {
		t.Errorf("room-state direction = %q, want %q", roomState.Direction, DirectionReceive)
	}
	if roomState.IsCommand() {
		t.Error("room-state is a command, want an event")
	}
	if roomState.DefName != "RoomState" || roomState.PayloadType != "RoomState" {
		t.Errorf("room-state types = %q/%q, want RoomState/RoomState", roomState.DefName, roomState.PayloadType)
	}
	if roomState.SchemaFile != "protocol/schemas/events.schema.json" || roomState.SchemaPointer != "#/$defs/RoomState" {
		t.Errorf("room-state schema = %q%s", roomState.SchemaFile, roomState.SchemaPointer)
	}

	reorder := messageByType(t, spec, "reorder-story")
	if reorder.Direction != DirectionSend || !reorder.IsCommand() {
		t.Errorf("reorder-story direction = %q, want send", reorder.Direction)
	}
	if reorder.DefName != "ReorderStoryCommand" || reorder.PayloadType != "ReorderStoryPayload" {
		t.Errorf("reorder-story types = %q/%q", reorder.DefName, reorder.PayloadType)
	}

	guarded := spec.GuardedMessages()
	if len(guarded) != 12 {
		t.Fatalf("guarded commands = %d, want 12", len(guarded))
	}
	for _, wireType := range wireTypes(guarded) {
		if wireType == "vote" || wireType == "update-name" || wireType == "add-story" {
			t.Errorf("command %q must not be guarded", wireType)
		}
	}
}

func TestLoadSpecSchemaShape(t *testing.T) {
	spec := loadSampleSpec(t)

	story := definitionByName(t, spec, "Story").Schema
	result, ok := story.Property("result")
	if !ok {
		t.Fatal("Story.result is missing")
	}
	if result.Required {
		t.Error("Story.result must be optional")
	}
	if result.Schema.Kind != KindNumber || result.Schema.GoType != "float32" {
		t.Errorf("Story.result = %q/%q, want number/float32", result.Schema.Kind, result.Schema.GoType)
	}
	mostAppearingVotes, ok := story.Property("mostAppearingVotes")
	if !ok || !mostAppearingVotes.Schema.Nullable || !mostAppearingVotes.Required {
		t.Errorf("Story.mostAppearingVotes must be required and nullable")
	}

	participant := definitionByName(t, spec, "Participant").Schema
	vote, ok := participant.Property("vote")
	if !ok || !vote.Schema.Nullable || !vote.Required {
		t.Errorf("Participant.vote must be required and nullable")
	}
	votedAt, ok := participant.Property("votedAt")
	if !ok || votedAt.Schema.Format != "date-time" || votedAt.Required {
		t.Errorf("Participant.votedAt must be an optional date-time")
	}

	roomState := definitionByName(t, spec, "RoomState").Schema
	stories, ok := roomState.Property("stories")
	if !ok || !stories.Schema.Nullable || stories.Required || stories.Schema.Kind != KindArray {
		t.Fatalf("RoomState.stories must be an optional nullable array")
	}
	if stories.Schema.Items == nil || stories.Schema.Items.RefName != "Story" {
		t.Errorf("RoomState.stories items = %+v, want $ref Story", stories.Schema.Items)
	}
	roomVersion, ok := roomState.Property("roomVersion")
	if !ok || roomVersion.Schema.GoType != "uint64" {
		t.Errorf("RoomState.roomVersion must carry x-go-type uint64")
	}

	deck := definitionByName(t, spec, "DeckDescriptor").Schema
	kind, ok := deck.Property("kind")
	if !ok || kind.Schema.RefName != "DeckKind" {
		t.Errorf("DeckDescriptor.kind = %+v, want $ref DeckKind", kind)
	}
}

func TestPointerValue(t *testing.T) {
	document := map[string]any{
		"a": map[string]any{
			"b":    []any{"zero", "one"},
			"a/b":  "slash",
			"a~b":  "tilde",
			"deep": map[string]any{"value": "found"},
		},
	}

	tests := []struct {
		name     string
		pointer  string
		expected any
		wantErr  bool
	}{
		{name: "root", pointer: "", expected: document},
		{name: "nested", pointer: "/a/deep/value", expected: "found"},
		{name: "array index", pointer: "/a/b/1", expected: "one"},
		{name: "escaped slash", pointer: "/a/a~1b", expected: "slash"},
		{name: "escaped tilde", pointer: "/a/a~0b", expected: "tilde"},
		{name: "missing", pointer: "/a/missing", wantErr: true},
		{name: "array overflow", pointer: "/a/b/2", wantErr: true},
		{name: "scalar traversal", pointer: "/a/b/zero", wantErr: true},
		{name: "missing slash", pointer: "a", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := pointerValue(document, test.pointer)
			if test.wantErr {
				if err == nil {
					t.Fatalf("pointerValue(%q) = %v, want error", test.pointer, value)
				}
				return
			}
			if err != nil {
				t.Fatalf("pointerValue(%q): %v", test.pointer, err)
			}
			if test.pointer == "" {
				return
			}
			if value != test.expected {
				t.Fatalf("pointerValue(%q) = %v, want %v", test.pointer, value, test.expected)
			}
		})
	}
}

func TestLoadSpecRejectsDuplicateDefinitions(t *testing.T) {
	root := tempRepo(t)
	extra := `{"$defs": {"RoomState": {"type": "object", "properties": {}}}}`
	if err := os.WriteFile(filepath.Join(root, "protocol", "schemas", "extra.schema.json"), []byte(extra), 0o644); err != nil {
		t.Fatalf("write extra schema: %v", err)
	}

	_, err := loadSpec(root)
	if err == nil || !strings.Contains(err.Error(), "duplicate $defs") {
		t.Fatalf("loadSpec error = %v, want duplicate $defs", err)
	}
}

func TestLoadSpecRejectsUnsupportedAsyncAPIVersion(t *testing.T) {
	root := tempRepo(t)
	mutateFile(t, filepath.Join(root, "protocol", "asyncapi.yaml"), "asyncapi: 3.0.0", "asyncapi: 2.6.0")

	_, err := loadSpec(root)
	if err == nil || !strings.Contains(err.Error(), "must start with 3") {
		t.Fatalf("loadSpec error = %v, want asyncapi version rejection", err)
	}
}

func TestLoadSpecRejectsMissingChannelMessage(t *testing.T) {
	root := tempRepo(t)
	mutateFile(
		t,
		filepath.Join(root, "protocol", "asyncapi.yaml"),
		"      vote:\n        $ref: '#/components/messages/vote'\n",
		"",
	)

	_, err := loadSpec(root)
	if err == nil || !strings.Contains(err.Error(), "not exposed by channels.roomSocket") {
		t.Fatalf("loadSpec error = %v, want missing channel message rejection", err)
	}
}

func TestLoadSpecRejectsConstMismatch(t *testing.T) {
	root := tempRepo(t)
	mutateFile(t, filepath.Join(root, "protocol", "schemas", "commands.schema.json"), `"const": "vote"`, `"const": "vote-x"`)

	_, err := loadSpec(root)
	if err == nil || !strings.Contains(err.Error(), "type.const must equal") {
		t.Fatalf("loadSpec error = %v, want const mismatch rejection", err)
	}
}

func TestRunCheckDetectsModifiedArtifact(t *testing.T) {
	root := tempRepo(t)

	var stdout, stderr bytes.Buffer
	if status := runIn(root, []string{"--check"}, &stdout, &stderr); status != 1 {
		t.Fatalf("check before generation status = %d, want 1", status)
	}
	if !strings.Contains(stderr.String(), typesGoPath) {
		t.Fatalf("check output %q does not list %s", stderr.String(), typesGoPath)
	}

	stdout.Reset()
	stderr.Reset()
	if status := runIn(root, nil, &stdout, &stderr); status != 0 {
		t.Fatalf("generate status = %d, want 0 (stderr: %s)", status, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if status := runIn(root, []string{"--check"}, &stdout, &stderr); status != 0 {
		t.Fatalf("check after generation status = %d, want 0 (stderr: %s)", status, stderr.String())
	}
	if !strings.Contains(stdout.String(), "up to date") {
		t.Fatalf("check output = %q, want up to date", stdout.String())
	}

	target := filepath.Join(root, filepath.FromSlash(typesGoPath))
	file, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open generated file: %v", err)
	}
	if _, err := file.WriteString("\n// manual edit\n"); err != nil {
		t.Fatalf("append to generated file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close generated file: %v", err)
	}

	stderr.Reset()
	if status := runIn(root, []string{"--check"}, &stdout, &stderr); status != 1 {
		t.Fatalf("check after manual edit status = %d, want 1", status)
	}
	if !strings.Contains(stderr.String(), typesGoPath) {
		t.Fatalf("check output %q does not list %s", stderr.String(), typesGoPath)
	}
}

func TestFindRepoRoot(t *testing.T) {
	root := repoRoot(t)
	nested := filepath.Join(root, "tools", "protocolgen")
	found, err := findRepoRoot(nested)
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	if found != root {
		t.Fatalf("findRepoRoot = %q, want %q", found, root)
	}
}
