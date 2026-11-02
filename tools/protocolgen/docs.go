package main

import (
	"fmt"
	"strings"
)

const protocolReadmePath = "protocol/README.md"

// emitDocs renders protocol/README.md.
func emitDocs(spec *Spec) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString(generatedHeader)
	builder.WriteString("\n\n# Planning Poker WebSocket protocol\n\n")
	builder.WriteString("This directory is the single source of truth for the realtime room WebSocket contract.\n")
	builder.WriteString("The AsyncAPI document describes the channel, operations, and messages; the JSON Schema\n")
	builder.WriteString("files describe every payload. Go and TypeScript types and validators are generated from\n")
	builder.WriteString("these files and must not be edited by hand.\n\n")
	builder.WriteString("- AsyncAPI document: [asyncapi.yaml](asyncapi.yaml)\n")
	builder.WriteString("- Payload schemas: [schemas/](schemas/)\n\n")

	builder.WriteString("## Messages\n\n")
	builder.WriteString("| Wire type | Direction | Schema | Summary |\n")
	builder.WriteString("| --- | --- | --- | --- |\n")
	for _, message := range spec.Messages {
		direction := "server -> client"
		if message.IsCommand() {
			direction = "client -> server"
		}
		link := strings.TrimPrefix(message.SchemaFile, "protocol/")
		fmt.Fprintf(
			&builder,
			"| `%s` | %s | [`%s`](%s%s) | %s |\n",
			message.WireType,
			direction,
			message.DefName,
			link,
			message.SchemaPointer,
			message.Summary,
		)
	}

	builder.WriteString("\n## Guarded commands\n\n")
	builder.WriteString("Commands whose payload carries `expectedRoomVersion` are guarded: the client refuses to\n")
	builder.WriteString("send them until the room version is known, and the server rejects stale versions.\n\n")
	for _, message := range spec.GuardedMessages() {
		fmt.Fprintf(&builder, "- `%s`\n", message.WireType)
	}

	builder.WriteString("\n## Changing the protocol\n\n")
	builder.WriteString("1. Edit `protocol/asyncapi.yaml` and/or the JSON Schema files under `protocol/schemas/`.\n")
	builder.WriteString("2. Run `make protocol` to regenerate the committed artifacts.\n")
	builder.WriteString("3. Run `make protocol-check` to confirm the generated files match the spec.\n")
	builder.WriteString("4. Commit the spec change together with every regenerated file.\n\n")
	builder.WriteString("Generated artifacts:\n\n")
	builder.WriteString("- `internal/protocol/types.generated.go`\n")
	builder.WriteString("- `internal/protocol/messages.generated.go`\n")
	builder.WriteString("- `frontend/planning-poker-front/src/components/messages/protocol.generated.ts`\n")
	builder.WriteString("- `protocol/README.md`\n\n")
	builder.WriteString("Never hand-edit a generated file: the next `make protocol` run overwrites it.\n")

	return []byte(builder.String()), nil
}
