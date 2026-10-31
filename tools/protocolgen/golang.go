package main

import (
	"fmt"
	"go/format"
	"strings"
)

const goProtocolPackage = "protocol"

// emitGoTypes renders internal/protocol/types.generated.go.
func emitGoTypes(spec *Spec) ([]byte, error) {
	var header strings.Builder
	header.WriteString(generatedHeader)
	header.WriteString("\n\npackage ")
	header.WriteString(goProtocolPackage)
	header.WriteString("\n")

	var body strings.Builder
	for _, definition := range spec.Definitions {
		switch {
		case definition.Schema.Kind == KindObject:
			if err := emitGoStruct(&body, spec, definition); err != nil {
				return nil, err
			}
		case definition.Schema.Kind == KindString && len(definition.Schema.Enum) > 0:
			emitGoEnum(&body, definition)
		default:
			return nil, fmt.Errorf("$defs.%s: unsupported definition kind %q", definition.Name, definition.Schema.Kind)
		}
	}

	if specUsesTime(spec) {
		header.WriteString("\nimport \"time\"\n")
	}
	header.WriteString(body.String())

	return format.Source([]byte(header.String()))
}

func emitGoStruct(builder *strings.Builder, spec *Spec, definition *Definition) error {
	fmt.Fprintf(builder, "\ntype %s struct {\n", definition.Name)
	for _, property := range definition.Schema.Properties {
		typeName, omitempty, err := goPropertyType(spec, property)
		if err != nil {
			return fmt.Errorf("$defs.%s.%s: %w", definition.Name, property.Name, err)
		}
		tag := property.Name
		if omitempty {
			tag += ",omitempty"
		}
		fmt.Fprintf(builder, "\t%s %s `json:%s`\n", goName(property.Name), typeName, quote(tag))
	}
	builder.WriteString("}\n")
	return nil
}

func emitGoEnum(builder *strings.Builder, definition *Definition) {
	fmt.Fprintf(builder, "\ntype %s string\n", definition.Name)
	builder.WriteString("\nconst (\n")
	for _, value := range definition.Schema.Enum {
		fmt.Fprintf(builder, "\t%s%s %s = %s\n", definition.Name, goConstantSuffix(value), definition.Name, quote(value))
	}
	builder.WriteString(")\n")
}

// goPropertyType returns the Go field type and whether the JSON tag carries omitempty.
func goPropertyType(spec *Spec, property Property) (string, bool, error) {
	base, err := goTypeName(spec, property.Schema)
	if err != nil {
		return "", false, err
	}

	if property.Schema.Kind == KindArray {
		// Slices are emitted as-is: optional and nullable slices both carry explicit null.
		return base, false, nil
	}
	if property.Schema.Nullable || !property.Required {
		return "*" + base, !property.Required, nil
	}
	return base, false, nil
}

func goTypeName(spec *Spec, schema *Schema) (string, error) {
	switch schema.Kind {
	case KindRef:
		if _, ok := spec.Definition(schema.RefName); !ok {
			return "", fmt.Errorf("unknown $ref %q", schema.RefName)
		}
		return schema.RefName, nil
	case KindString:
		if schema.Format == "date-time" {
			return "time.Time", nil
		}
		return "string", nil
	case KindInteger:
		return goIntegerType(schema.GoType)
	case KindNumber:
		return goNumberType(schema.GoType)
	case KindBoolean:
		return "bool", nil
	case KindArray:
		return goArrayType(spec, schema)
	default:
		return "", fmt.Errorf("unsupported type %q", schema.Kind)
	}
}

func goIntegerType(goType string) (string, error) {
	switch goType {
	case "":
		return "int", nil
	case "uint64":
		return "uint64", nil
	default:
		return "", fmt.Errorf("unsupported x-go-type %q for integer", goType)
	}
}

func goNumberType(goType string) (string, error) {
	switch goType {
	case "":
		return "float64", nil
	case "float32":
		return "float32", nil
	default:
		return "", fmt.Errorf("unsupported x-go-type %q for number", goType)
	}
}

func goArrayType(spec *Spec, schema *Schema) (string, error) {
	if schema.Items == nil {
		return "", fmt.Errorf("array without items")
	}
	itemType, err := goTypeName(spec, schema.Items)
	if err != nil {
		return "", err
	}
	if schema.Items.Nullable {
		itemType = "*" + itemType
	}
	return "[]" + itemType, nil
}

func specUsesTime(spec *Spec) bool {
	for _, definition := range spec.Definitions {
		if schemaUsesTime(spec, definition.Schema) {
			return true
		}
	}
	return false
}

func schemaUsesTime(spec *Spec, schema *Schema) bool {
	switch schema.Kind {
	case KindObject:
		for _, property := range schema.Properties {
			if schemaUsesTime(spec, property.Schema) {
				return true
			}
		}
		return false
	case KindString:
		return schema.Format == "date-time"
	case KindArray:
		return schema.Items != nil && schemaUsesTime(spec, schema.Items)
	case KindRef:
		definition, ok := spec.Definition(schema.RefName)
		return ok && schemaUsesTime(spec, definition.Schema)
	default:
		return false
	}
}

// emitGoMessages renders internal/protocol/messages.generated.go.
func emitGoMessages(spec *Spec) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString(generatedHeader)
	builder.WriteString("\n\npackage ")
	builder.WriteString(goProtocolPackage)
	builder.WriteString("\n\nconst (\n")
	for _, message := range spec.Messages {
		fmt.Fprintf(&builder, "\tMessageType%s = %s\n", goName(message.WireType), quote(message.WireType))
	}
	builder.WriteString(")\n")

	builder.WriteString("\nvar (\n")
	builder.WriteString("\tClientCommandTypes = []string{\n")
	for _, message := range spec.CommandMessages() {
		fmt.Fprintf(&builder, "\t\t%s,\n", quote(message.WireType))
	}
	builder.WriteString("\t}\n\n\tServerEventTypes = []string{\n")
	for _, message := range spec.EventMessages() {
		fmt.Fprintf(&builder, "\t\t%s,\n", quote(message.WireType))
	}
	builder.WriteString("\t}\n)\n")

	builder.WriteString("\n// IsClientCommandType reports whether the wire type is a client command.\n")
	builder.WriteString("func IsClientCommandType(messageType string) bool {\n")
	for _, message := range spec.CommandMessages() {
		fmt.Fprintf(&builder, "\tif messageType == MessageType%s {\n\t\treturn true\n\t}\n", goName(message.WireType))
	}
	builder.WriteString("\treturn false\n}\n")

	builder.WriteString("\n// IsServerEventType reports whether the wire type is a server event.\n")
	builder.WriteString("func IsServerEventType(messageType string) bool {\n")
	for _, message := range spec.EventMessages() {
		fmt.Fprintf(&builder, "\tif messageType == MessageType%s {\n\t\treturn true\n\t}\n", goName(message.WireType))
	}
	builder.WriteString("\treturn false\n}\n")

	builder.WriteString("\n// NewClientCommandPayload returns a pointer to the zero payload struct for a command.\n")
	builder.WriteString("func NewClientCommandPayload(messageType string) (any, bool) {\n")
	builder.WriteString("\tswitch messageType {\n")
	for _, message := range spec.CommandMessages() {
		fmt.Fprintf(&builder, "\tcase MessageType%s:\n\t\treturn &%s{}, true\n", goName(message.WireType), message.PayloadType)
	}
	builder.WriteString("\t}\n\treturn nil, false\n}\n")

	return format.Source([]byte(builder.String()))
}
