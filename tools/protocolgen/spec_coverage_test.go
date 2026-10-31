package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func sampleLoader(t *testing.T) (*schemaLoader, string) {
	t.Helper()
	root := tempRepo(t)
	loader, err := newSchemaLoader(root)
	if err != nil {
		t.Fatalf("newSchemaLoader: %v", err)
	}
	return loader, filepath.Join(root, filepath.FromSlash(asyncAPIYAMLPath))
}

func TestNormalizeValue(t *testing.T) {
	input := map[string]any{
		"scalar": "value",
		"nested": map[any]any{"key": []any{map[string]any{"inner": 1}}},
	}
	want := map[string]any{
		"scalar": "value",
		"nested": map[string]any{"key": []any{map[string]any{"inner": 1}}},
	}
	got, err := normalizeValue(input)
	if err != nil {
		t.Fatalf("normalizeValue: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeValue = %#v, want %#v", got, want)
	}

	if got, err := normalizeValue(42); err != nil || got != 42 {
		t.Fatalf("normalizeValue(42) = %v/%v, want 42/nil", got, err)
	}

	for name, value := range map[string]any{
		"map":   map[string]any{"bad": map[any]any{1: "value"}},
		"slice": []any{map[any]any{1: "value"}},
	} {
		t.Run("non-string-key-in-"+name, func(t *testing.T) {
			if _, err := normalizeValue(value); err == nil || !strings.Contains(err.Error(), "non-string mapping key") {
				t.Fatalf("normalizeValue error = %v, want non-string mapping key", err)
			}
		})
	}
}

func TestSchemaKindResolvesTypes(t *testing.T) {
	tests := []struct {
		name     string
		raw      map[string]any
		wantKind SchemaKind
		wantNull bool
	}{
		{name: "object", raw: map[string]any{"type": "object"}, wantKind: KindObject},
		{name: "nullable list", raw: map[string]any{"type": []any{"string", "null"}}, wantKind: KindString, wantNull: true},
		{name: "null first", raw: map[string]any{"type": []any{"null", "integer"}}, wantKind: KindInteger, wantNull: true},
		{name: "enum without type", raw: map[string]any{"enum": []any{"a"}}, wantKind: KindString},
		{name: "const without type", raw: map[string]any{"const": "a"}, wantKind: KindString},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kind, nullable, err := schemaKind(test.raw)
			if err != nil {
				t.Fatalf("schemaKind: %v", err)
			}
			if kind != test.wantKind || nullable != test.wantNull {
				t.Fatalf("schemaKind = %q/%v, want %q/%v", kind, nullable, test.wantKind, test.wantNull)
			}
		})
	}
}

func TestSchemaKindRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		raw     map[string]any
		wantErr string
	}{
		{name: "missing type", raw: map[string]any{}, wantErr: "missing type"},
		{name: "non-string type", raw: map[string]any{"type": 1}, wantErr: "type must be a string or an array of strings"},
		{name: "only null", raw: map[string]any{"type": []any{"null"}}, wantErr: "type must contain a non-null entry"},
		{name: "two types", raw: map[string]any{"type": []any{"string", "integer"}}, wantErr: "only one non-null type is supported"},
		{name: "non-string entry", raw: map[string]any{"type": []any{1}}, wantErr: "type entries must be strings"},
		{name: "unsupported scalar", raw: map[string]any{"type": "map"}, wantErr: "unsupported type"},
		{name: "unsupported entry", raw: map[string]any{"type": []any{"map"}}, wantErr: "unsupported type"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := schemaKind(test.raw); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("schemaKind error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestSchemaKindName(t *testing.T) {
	for _, name := range []string{"object", "string", "integer", "number", "boolean", "array"} {
		kind, err := schemaKindName(name)
		if err != nil || string(kind) != name {
			t.Fatalf("schemaKindName(%q) = %q/%v, want %q", name, kind, err, name)
		}
	}
	if _, err := schemaKindName("null"); err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("schemaKindName(null) error = %v, want unsupported type", err)
	}
}

func TestSplitRefAndPointerName(t *testing.T) {
	tests := []struct {
		ref          string
		wantFile     string
		wantFragment string
	}{
		{ref: "events.schema.json#/$defs/RoomState", wantFile: "events.schema.json", wantFragment: "/$defs/RoomState"},
		{ref: "#/$defs/Vote", wantFile: "", wantFragment: "/$defs/Vote"},
		{ref: "events.schema.json", wantFile: "events.schema.json", wantFragment: ""},
		{ref: "events.schema.json#", wantFile: "events.schema.json", wantFragment: ""},
	}
	for _, test := range tests {
		t.Run(test.ref, func(t *testing.T) {
			file, fragment := splitRef(test.ref)
			if file != test.wantFile || fragment != test.wantFragment {
				t.Fatalf("splitRef(%q) = %q/%q, want %q/%q", test.ref, file, fragment, test.wantFile, test.wantFragment)
			}
		})
	}

	if got := pointerName(""); got != "" {
		t.Fatalf("pointerName(\"\") = %q, want empty", got)
	}
	if got := pointerName("/$defs/a~1b~0c"); got != "a/b~c" {
		t.Fatalf("pointerName escaped = %q, want a/b~c", got)
	}
}

func TestEntryRef(t *testing.T) {
	ref, err := entryRef(map[string]any{"$ref": "#/components/messages/vote"})
	if err != nil || ref != "vote" {
		t.Fatalf("entryRef = %q/%v, want vote", ref, err)
	}
	if _, err := entryRef("not-object"); err == nil || !strings.Contains(err.Error(), "entry must be an object") {
		t.Fatalf("entryRef error = %v, want object rejection", err)
	}
	if _, err := entryRef(map[string]any{}); err == nil || !strings.Contains(err.Error(), "entry requires a $ref") {
		t.Fatalf("entryRef error = %v, want $ref requirement", err)
	}
}

func TestOperationDirections(t *testing.T) {
	document := map[string]any{
		"operations": map[string]any{
			"sendCommand": map[string]any{
				"action":   "send",
				"messages": []any{map[string]any{"$ref": "#/components/messages/vote"}},
			},
			"receiveEvent": map[string]any{
				"action":   "receive",
				"messages": []any{map[string]any{"$ref": "#/components/messages/room-state"}},
			},
		},
	}
	directions, err := operationDirections(document)
	if err != nil {
		t.Fatalf("operationDirections: %v", err)
	}
	if directions["vote"] != DirectionSend || directions["room-state"] != DirectionReceive {
		t.Fatalf("operationDirections = %v, want vote/send and room-state/receive", directions)
	}

	message := func(ref string) any { return map[string]any{"$ref": ref} }
	tests := []struct {
		name     string
		document map[string]any
		wantErr  string
	}{
		{name: "missing operations", document: map[string]any{}, wantErr: "missing operations object"},
		{name: "operation not object", document: map[string]any{"operations": map[string]any{"send": "nope"}}, wantErr: "must be an object"},
		{name: "invalid action", document: map[string]any{"operations": map[string]any{"send": map[string]any{"action": "publish", "messages": []any{message("#/m/a")}}}}, wantErr: "action must be send or receive"},
		{name: "messages not array", document: map[string]any{"operations": map[string]any{"send": map[string]any{"action": "send", "messages": "nope"}}}, wantErr: "must be a non-empty array"},
		{name: "messages empty", document: map[string]any{"operations": map[string]any{"send": map[string]any{"action": "send", "messages": []any{}}}}, wantErr: "must be a non-empty array"},
		{name: "entry not object", document: map[string]any{"operations": map[string]any{"send": map[string]any{"action": "send", "messages": []any{"nope"}}}}, wantErr: "entry must be an object"},
		{name: "duplicate message", document: map[string]any{"operations": map[string]any{
			"send":    map[string]any{"action": "send", "messages": []any{message("#/m/a")}},
			"receive": map[string]any{"action": "receive", "messages": []any{message("#/m/a")}},
		}}, wantErr: "listed by multiple operations"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := operationDirections(test.document); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("operationDirections error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestResolveRef(t *testing.T) {
	loader, _ := sampleLoader(t)
	commandsFile := filepath.Join(loader.root, filepath.FromSlash(schemaDirectory), "commands.schema.json")
	eventsFile := filepath.Join(loader.root, filepath.FromSlash(schemaDirectory), "events.schema.json")

	target, fragment, err := loader.resolveRef(commandsFile, "#/$defs/VotePayload")
	if err != nil || fragment != "/$defs/VotePayload" || filepath.Clean(target) != filepath.Clean(commandsFile) {
		t.Fatalf("resolveRef local = %q/%q/%v", target, fragment, err)
	}

	target, fragment, err = loader.resolveRef(commandsFile, "events.schema.json#/$defs/RoomState")
	if err != nil || fragment != "/$defs/RoomState" || filepath.Clean(target) != filepath.Clean(eventsFile) {
		t.Fatalf("resolveRef relative = %q/%q/%v", target, fragment, err)
	}

	target, fragment, err = loader.resolveRef(commandsFile, filepath.ToSlash(eventsFile)+"#/$defs/RoomState")
	if err != nil || fragment != "/$defs/RoomState" || filepath.Clean(target) != filepath.Clean(eventsFile) {
		t.Fatalf("resolveRef absolute = %q/%q/%v", target, fragment, err)
	}

	if _, _, err := loader.resolveRef(commandsFile, "missing.schema.json#/$defs/RoomState"); err == nil || !strings.Contains(err.Error(), "resolve $ref") {
		t.Fatalf("resolveRef error = %v, want load failure", err)
	}
	if _, _, err := loader.resolveRef(commandsFile, "#/$defs/Missing"); err == nil || !strings.Contains(err.Error(), "pointer segment") {
		t.Fatalf("resolveRef error = %v, want pointer failure", err)
	}
}

func TestLoadDoc(t *testing.T) {
	loader, _ := sampleLoader(t)
	file := filepath.Join(loader.root, filepath.FromSlash(schemaDirectory), "events.schema.json")

	document, err := loader.loadDoc(file)
	if err != nil {
		t.Fatalf("loadDoc: %v", err)
	}
	cached, err := loader.loadDoc(file)
	if err != nil || !reflect.DeepEqual(document, cached) {
		t.Fatalf("loadDoc cached = %#v/%v, want %#v", cached, err, document)
	}

	if _, err := loader.loadDoc(filepath.Join(loader.root, filepath.FromSlash(schemaDirectory), "missing.schema.json")); err == nil || !strings.Contains(err.Error(), "read schema") {
		t.Fatalf("loadDoc error = %v, want read failure", err)
	}

	invalid := filepath.Join(loader.root, filepath.FromSlash(schemaDirectory), "invalid.schema.json")
	if err := os.WriteFile(invalid, []byte("{"), 0o644); err != nil {
		t.Fatalf("write invalid schema: %v", err)
	}
	if _, err := loader.loadDoc(invalid); err == nil || !strings.Contains(err.Error(), "parse schema") {
		t.Fatalf("loadDoc error = %v, want parse failure", err)
	}
}

func TestParseDefinitionUnknownAndCircular(t *testing.T) {
	loader, _ := sampleLoader(t)
	if _, err := loader.parseDefinition("Missing"); err == nil || !strings.Contains(err.Error(), "unknown $defs") {
		t.Fatalf("parseDefinition error = %v, want unknown $defs", err)
	}

	root := tempRepo(t)
	circular := `{"$defs": {"CycleA": {"$ref": "#/$defs/CycleB"}, "CycleB": {"$ref": "#/$defs/CycleA"}}}`
	if err := os.WriteFile(filepath.Join(root, "protocol", "schemas", "cycle.schema.json"), []byte(circular), 0o644); err != nil {
		t.Fatalf("write circular schema: %v", err)
	}
	if _, err := newSchemaLoader(root); err == nil || !strings.Contains(err.Error(), "circular $ref") {
		t.Fatalf("newSchemaLoader error = %v, want circular $ref", err)
	}
}

func TestSchemaFilesErrors(t *testing.T) {
	if _, err := schemaFiles(filepath.Join(t.TempDir(), "missing")); err == nil || !strings.Contains(err.Error(), "read schema directory") {
		t.Fatalf("schemaFiles error = %v, want read failure", err)
	}
	if _, err := schemaFiles(t.TempDir()); err == nil || !strings.Contains(err.Error(), "no *.schema.json files") {
		t.Fatalf("schemaFiles error = %v, want empty rejection", err)
	}
}

func TestNewSchemaLoaderRejectsMissingDefs(t *testing.T) {
	root := tempRepo(t)
	if err := os.WriteFile(filepath.Join(root, "protocol", "schemas", "nodefs.schema.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	if _, err := newSchemaLoader(root); err == nil || !strings.Contains(err.Error(), "missing $defs object") {
		t.Fatalf("newSchemaLoader error = %v, want missing $defs", err)
	}
}

func TestNewSchemaLoaderRejectsNonObjectDefinition(t *testing.T) {
	root := tempRepo(t)
	if err := os.WriteFile(filepath.Join(root, "protocol", "schemas", "baddef.schema.json"), []byte(`{"$defs": {"Bad": 1}}`), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	if _, err := newSchemaLoader(root); err == nil || !strings.Contains(err.Error(), "is not an object") {
		t.Fatalf("newSchemaLoader error = %v, want definition rejection", err)
	}
}

func TestApplySchemaMetadataErrors(t *testing.T) {
	if err := applySchemaMetadata(&Schema{Kind: KindString}, map[string]any{"format": 1}); err == nil || !strings.Contains(err.Error(), "format must be a string") {
		t.Fatalf("format error = %v, want type rejection", err)
	}
	if err := applySchemaMetadata(&Schema{Kind: KindString}, map[string]any{"x-go-type": 1}); err == nil || !strings.Contains(err.Error(), "x-go-type must be a string") {
		t.Fatalf("x-go-type error = %v, want type rejection", err)
	}

	schema := &Schema{Kind: KindString}
	if err := applySchemaMetadata(schema, map[string]any{"format": "uuid", "x-go-type": "string"}); err != nil {
		t.Fatalf("applySchemaMetadata: %v", err)
	}
	if schema.Format != "uuid" || schema.GoType != "string" {
		t.Fatalf("schema metadata = %q/%q, want uuid/string", schema.Format, schema.GoType)
	}
}

func TestApplyEnumErrors(t *testing.T) {
	if err := applyEnum(&Schema{Kind: KindString}, "nope"); err == nil || !strings.Contains(err.Error(), "enum must be an array") {
		t.Fatalf("applyEnum error = %v, want array rejection", err)
	}
	if err := applyEnum(&Schema{Kind: KindInteger}, []any{"a"}); err == nil || !strings.Contains(err.Error(), "enum requires a string type") {
		t.Fatalf("applyEnum error = %v, want kind rejection", err)
	}
	if err := applyEnum(&Schema{Kind: KindString}, []any{1}); err == nil || !strings.Contains(err.Error(), "enum values must be strings") {
		t.Fatalf("applyEnum error = %v, want value rejection", err)
	}
	if err := applyEnum(&Schema{Kind: KindString}, []any{"a", "a"}); err == nil || !strings.Contains(err.Error(), "duplicate enum value") {
		t.Fatalf("applyEnum error = %v, want duplicate rejection", err)
	}
	if err := applyEnum(&Schema{Kind: KindString}, []any{}); err == nil || !strings.Contains(err.Error(), "enum must not be empty") {
		t.Fatalf("applyEnum error = %v, want empty rejection", err)
	}
	schema := &Schema{Kind: KindString}
	if err := applyEnum(schema, []any{"a", "b"}); err != nil || !reflect.DeepEqual(schema.Enum, []string{"a", "b"}) {
		t.Fatalf("applyEnum = %v/%v, want [a b]", schema.Enum, err)
	}
}

func TestApplyConstErrors(t *testing.T) {
	if err := applyConst(&Schema{Kind: KindString}, 1); err == nil || !strings.Contains(err.Error(), "const must be a string") {
		t.Fatalf("applyConst error = %v, want type rejection", err)
	}
	if err := applyConst(&Schema{Kind: KindInteger}, "a"); err == nil || !strings.Contains(err.Error(), "const requires a string type") {
		t.Fatalf("applyConst error = %v, want kind rejection", err)
	}
	schema := &Schema{Kind: KindString}
	if err := applyConst(schema, "value"); err != nil || schema.Const != "value" {
		t.Fatalf("applyConst = %q/%v, want value", schema.Const, err)
	}
}

func TestParseObjectAndArrayErrors(t *testing.T) {
	loader, _ := sampleLoader(t)
	file := filepath.Join(loader.root, filepath.FromSlash(schemaDirectory), "commands.schema.json")

	tests := []struct {
		name    string
		raw     map[string]any
		wantErr string
	}{
		{name: "properties not object", raw: map[string]any{"type": "object", "properties": []any{}}, wantErr: "properties must be an object"},
		{name: "required undeclared", raw: map[string]any{"type": "object", "properties": map[string]any{}, "required": []any{"missing"}}, wantErr: "is not declared"},
		{name: "property not object", raw: map[string]any{"type": "object", "properties": map[string]any{"value": 1}}, wantErr: "must be an object"},
		{name: "array without items", raw: map[string]any{"type": "array"}, wantErr: "array items are required"},
		{name: "array items not object", raw: map[string]any{"type": "array", "items": 1}, wantErr: "array items must be an object"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := loader.parseSchema(test.raw, file); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("parseSchema error = %v, want %q", err, test.wantErr)
			}
		})
	}

	schema, err := loader.parseSchema(map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, file)
	if err != nil || schema.Items == nil || schema.Items.Kind != KindString {
		t.Fatalf("parseSchema array items = %+v/%v", schema, err)
	}
}

func TestRequiredNamesErrors(t *testing.T) {
	if _, err := requiredNames(map[string]any{"required": "nope"}); err == nil || !strings.Contains(err.Error(), "required must be an array") {
		t.Fatalf("requiredNames error = %v, want array rejection", err)
	}
	if _, err := requiredNames(map[string]any{"required": []any{1}}); err == nil || !strings.Contains(err.Error(), "required entries must be strings") {
		t.Fatalf("requiredNames error = %v, want entry rejection", err)
	}
	required, err := requiredNames(map[string]any{"required": []any{"a", "b"}})
	if err != nil || !required["a"] || !required["b"] {
		t.Fatalf("requiredNames = %v/%v, want a and b", required, err)
	}
}

func TestParseRefSchemaErrors(t *testing.T) {
	loader, _ := sampleLoader(t)
	commandsFile := filepath.Join(loader.root, filepath.FromSlash(schemaDirectory), "commands.schema.json")

	if _, err := loader.parseSchema(map[string]any{"$ref": 1}, commandsFile); err == nil || !strings.Contains(err.Error(), "$ref must be a string") {
		t.Fatalf("parseSchema error = %v, want $ref type rejection", err)
	}
	if _, err := loader.parseSchema(map[string]any{"$ref": "#/$schema"}, commandsFile); err == nil || !strings.Contains(err.Error(), "does not point at a known $defs entry") {
		t.Fatalf("parseSchema error = %v, want unknown ref rejection", err)
	}
	loader.defs["$schema"] = &rawDefinition{
		name:    "$schema",
		file:    filepath.Join(loader.root, filepath.FromSlash(schemaDirectory), "events.schema.json"),
		relFile: "protocol/schemas/events.schema.json",
	}
	if _, err := loader.parseRefSchema("#/$schema", commandsFile); err == nil || !strings.Contains(err.Error(), "lives in") {
		t.Fatalf("parseRefSchema error = %v, want cross-file rejection", err)
	}
	ref, err := loader.parseRefSchema("#/$defs/VotePayload", commandsFile)
	if err != nil || ref.Kind != KindRef || ref.RefName != "VotePayload" {
		t.Fatalf("parseRefSchema = %+v/%v, want VotePayload ref", ref, err)
	}
}

func TestValidateMessageExposure(t *testing.T) {
	entry := func(ref string) any { return map[string]any{"$ref": ref} }
	tests := []struct {
		name     string
		messages map[string]any
		channels map[string]any
		wantErr  string
	}{
		{
			name:     "channel message missing",
			messages: map[string]any{},
			channels: map[string]any{"vote": entry("#/components/messages/vote")},
			wantErr:  "missing from components.messages",
		},
		{
			name:     "invalid channel entry",
			messages: map[string]any{"vote": map[string]any{}},
			channels: map[string]any{"vote": "nope"},
			wantErr:  "entry must be an object",
		},
		{
			name:     "ref mismatch",
			messages: map[string]any{"vote": map[string]any{}, "other": map[string]any{}},
			channels: map[string]any{"vote": entry("#/components/messages/other")},
			wantErr:  "references",
		},
		{
			name:     "component not exposed",
			messages: map[string]any{"vote": map[string]any{}, "other": map[string]any{}},
			channels: map[string]any{"vote": entry("#/components/messages/vote")},
			wantErr:  "is not exposed by channels.roomSocket",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateMessageExposure(test.messages, test.channels); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("validateMessageExposure error = %v, want %q", err, test.wantErr)
			}
		})
	}

	messages := map[string]any{"vote": map[string]any{}}
	channels := map[string]any{"vote": entry("#/components/messages/vote")}
	if err := validateMessageExposure(messages, channels); err != nil {
		t.Fatalf("validateMessageExposure: %v", err)
	}
}

func TestReadAsyncAPIDocumentErrors(t *testing.T) {
	if _, err := readAsyncAPIDocument(t.TempDir()); err == nil || !strings.Contains(err.Error(), "read protocol/asyncapi.yaml") {
		t.Fatalf("readAsyncAPIDocument error = %v, want read failure", err)
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "protocol"), 0o755); err != nil {
		t.Fatalf("mkdir protocol: %v", err)
	}
	asyncAPI := filepath.Join(root, "protocol", "asyncapi.yaml")

	if err := os.WriteFile(asyncAPI, []byte("asyncapi: ["), 0o644); err != nil {
		t.Fatalf("write asyncapi: %v", err)
	}
	if _, err := readAsyncAPIDocument(root); err == nil || !strings.Contains(err.Error(), "parse protocol/asyncapi.yaml") {
		t.Fatalf("readAsyncAPIDocument error = %v, want parse failure", err)
	}

	if err := os.WriteFile(asyncAPI, []byte("asyncapi: 2.6.0\n"), 0o644); err != nil {
		t.Fatalf("write asyncapi: %v", err)
	}
	if _, err := readAsyncAPIDocument(root); err == nil || !strings.Contains(err.Error(), "must start with 3") {
		t.Fatalf("readAsyncAPIDocument error = %v, want version rejection", err)
	}

	if err := os.WriteFile(asyncAPI, []byte("asyncapi: 3.0.0\n"), 0o644); err != nil {
		t.Fatalf("write asyncapi: %v", err)
	}
	document, err := readAsyncAPIDocument(root)
	if err != nil || document["asyncapi"] != "3.0.0" {
		t.Fatalf("readAsyncAPIDocument = %v/%v", document, err)
	}
}

func TestAsyncAPIMessagesErrors(t *testing.T) {
	if _, _, err := asyncAPIMessages(map[string]any{}); err == nil || !strings.Contains(err.Error(), "missing components object") {
		t.Fatalf("asyncAPIMessages error = %v, want components rejection", err)
	}
	if _, _, err := asyncAPIMessages(map[string]any{"components": map[string]any{}}); err == nil || !strings.Contains(err.Error(), "missing components.messages") {
		t.Fatalf("asyncAPIMessages error = %v, want messages rejection", err)
	}
	if _, _, err := asyncAPIMessages(map[string]any{"components": map[string]any{"messages": map[string]any{}}}); err == nil || !strings.Contains(err.Error(), "missing components.messages") {
		t.Fatalf("asyncAPIMessages error = %v, want empty messages rejection", err)
	}
	if _, _, err := asyncAPIMessages(map[string]any{"components": map[string]any{"messages": map[string]any{"vote": map[string]any{}}}}); err == nil || !strings.Contains(err.Error(), "missing channels object") {
		t.Fatalf("asyncAPIMessages error = %v, want channels rejection", err)
	}
}

func TestRoomChannelMessagesErrors(t *testing.T) {
	if _, err := roomChannelMessages(map[string]any{}); err == nil || !strings.Contains(err.Error(), "missing channels object") {
		t.Fatalf("roomChannelMessages error = %v, want channels rejection", err)
	}
	if _, err := roomChannelMessages(map[string]any{"channels": map[string]any{}}); err == nil || !strings.Contains(err.Error(), "missing channels.roomSocket") {
		t.Fatalf("roomChannelMessages error = %v, want roomSocket rejection", err)
	}
	if _, err := roomChannelMessages(map[string]any{"channels": map[string]any{"roomSocket": map[string]any{}}}); err == nil || !strings.Contains(err.Error(), "missing channels.roomSocket.messages") {
		t.Fatalf("roomChannelMessages error = %v, want messages rejection", err)
	}
	if _, err := roomChannelMessages(map[string]any{"channels": map[string]any{"roomSocket": map[string]any{"messages": map[string]any{}}}}); err == nil || !strings.Contains(err.Error(), "missing channels.roomSocket.messages") {
		t.Fatalf("roomChannelMessages error = %v, want empty messages rejection", err)
	}
}

func TestBuildMessageErrors(t *testing.T) {
	loader, asyncAPIFile := sampleLoader(t)
	directions := map[string]Direction{"vote": DirectionSend}
	validMessage := func() map[string]any {
		return map[string]any{
			"name":    "vote",
			"title":   "Vote",
			"summary": "Cast a vote.",
			"payload": map[string]any{"$ref": "schemas/commands.schema.json#/$defs/VoteCommand"},
		}
	}

	tests := []struct {
		name    string
		raw     any
		dirs    map[string]Direction
		wantErr string
	}{
		{name: "message not object", raw: "nope", dirs: directions, wantErr: "must be an object"},
		{name: "name mismatch", raw: func() map[string]any { m := validMessage(); m["name"] = "other"; return m }(), dirs: directions, wantErr: "name must be"},
		{name: "missing title", raw: func() map[string]any { m := validMessage(); delete(m, "title"); return m }(), dirs: directions, wantErr: "requires title and summary"},
		{name: "payload not object", raw: func() map[string]any { m := validMessage(); m["payload"] = "nope"; return m }(), dirs: directions, wantErr: "payload must be an object"},
		{name: "payload ref missing", raw: func() map[string]any { m := validMessage(); m["payload"] = map[string]any{}; return m }(), dirs: directions, wantErr: "payload.$ref is required"},
		{name: "payload resolve failure", raw: func() map[string]any {
			m := validMessage()
			m["payload"] = map[string]any{"$ref": "schemas/commands.schema.json#/$defs/Missing"}
			return m
		}(), dirs: directions, wantErr: "payload:"},
		{name: "payload definition not object", raw: func() map[string]any {
			m := validMessage()
			m["payload"] = map[string]any{"$ref": "schemas/shared.schema.json#/$defs/DeckKind"}
			return m
		}(), dirs: directions, wantErr: "must be an object"},
		{name: "missing direction", raw: validMessage(), dirs: map[string]Direction{}, wantErr: "not listed by any operation"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := buildMessage(loader, asyncAPIFile, "vote", test.raw, test.dirs); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("buildMessage error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestResolveMessagePayloadNotParsed(t *testing.T) {
	loader, asyncAPIFile := sampleLoader(t)
	delete(loader.parsed, "VoteCommand")
	message := map[string]any{"payload": map[string]any{"$ref": "schemas/commands.schema.json#/$defs/VoteCommand"}}
	if _, _, _, err := resolveMessagePayload(loader, asyncAPIFile, "vote", message); err == nil || !strings.Contains(err.Error(), "was not parsed") {
		t.Fatalf("resolveMessagePayload error = %v, want unparsed rejection", err)
	}
}

func TestPayloadTypeForErrors(t *testing.T) {
	loader := &schemaLoader{
		defs:   map[string]*rawDefinition{"Payload": {name: "Payload"}},
		parsed: map[string]*Schema{"Payload": {Kind: KindString}},
	}
	payloadSchema := &Schema{
		Kind: KindObject,
		Properties: []Property{
			{Name: "payload", Schema: &Schema{Kind: KindRef, RefName: "Payload"}},
		},
	}
	if _, err := payloadTypeFor(loader, payloadSchema, "VoteCommand", "vote", DirectionSend); err == nil || !strings.Contains(err.Error(), "must be an object") {
		t.Fatalf("payloadTypeFor error = %v, want object rejection", err)
	}

	unknown := &Schema{
		Kind: KindObject,
		Properties: []Property{
			{Name: "payload", Schema: &Schema{Kind: KindRef, RefName: "Missing"}},
		},
	}
	if _, err := payloadTypeFor(loader, unknown, "VoteCommand", "vote", DirectionSend); err == nil || !strings.Contains(err.Error(), "payload \"Missing\" is unknown") {
		t.Fatalf("payloadTypeFor error = %v, want unknown payload rejection", err)
	}

	missingRef := &Schema{Kind: KindObject}
	if _, err := payloadTypeFor(loader, missingRef, "VoteCommand", "vote", DirectionSend); err == nil || !strings.Contains(err.Error(), "must declare a payload $ref") {
		t.Fatalf("payloadTypeFor error = %v, want missing ref rejection", err)
	}

	if payloadType, err := payloadTypeFor(loader, missingRef, "RoomState", "room-state", DirectionReceive); err != nil || payloadType != "RoomState" {
		t.Fatalf("payloadTypeFor receive = %q/%v, want RoomState", payloadType, err)
	}
}

func TestBuildMessagesRejectsUnexposedOperation(t *testing.T) {
	loader, asyncAPIFile := sampleLoader(t)
	messagesRaw := map[string]any{
		"vote": map[string]any{
			"name":    "vote",
			"title":   "Vote",
			"summary": "Cast a vote.",
			"payload": map[string]any{"$ref": "schemas/commands.schema.json#/$defs/VoteCommand"},
		},
	}
	channelMessages := map[string]any{"vote": map[string]any{"$ref": "#/components/messages/vote"}}
	directions := map[string]Direction{"vote": DirectionSend, "room-state": DirectionReceive}
	if _, err := buildMessages(loader, asyncAPIFile, messagesRaw, channelMessages, directions); err == nil || !strings.Contains(err.Error(), "not exposed by channels.roomSocket") {
		t.Fatalf("buildMessages error = %v, want unexposed operation rejection", err)
	}
}

func TestNewSpecRejectsDuplicateDefinitions(t *testing.T) {
	_, err := newSpec(nil, []*Definition{{Name: "Duplicate"}, {Name: "Duplicate"}})
	if err == nil || !strings.Contains(err.Error(), "duplicate $defs") {
		t.Fatalf("newSpec error = %v, want duplicate rejection", err)
	}
}

func TestSpecGuardedMessagesSkipsMissingAndNonObjectDefinitions(t *testing.T) {
	spec := &Spec{
		Messages: []Message{
			{Direction: DirectionSend, PayloadType: "Missing"},
			{Direction: DirectionSend, PayloadType: "Scalar"},
		},
		byName: map[string]*Definition{"Scalar": {Name: "Scalar", Schema: &Schema{Kind: KindString}}},
	}
	if guarded := spec.GuardedMessages(); len(guarded) != 0 {
		t.Fatalf("GuardedMessages = %d, want 0", len(guarded))
	}
}
