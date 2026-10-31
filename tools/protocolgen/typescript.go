package main

import (
	"fmt"
	"strings"
)

const (
	typescriptOutputPath = "frontend/planning-poker-front/src/components/messages/protocol.generated.ts"
	runtimeImportPath    = "./protocol.runtime"
)

// emitTypeScript renders protocol.generated.ts.
func emitTypeScript(spec *Spec) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString(generatedHeader)
	builder.WriteString("\n\n")

	usage := collectTypeScriptUsage(spec)
	var helpers []string
	if usage.integer {
		helpers = append(helpers, "isInteger")
	}
	helpers = append(helpers, "isRecord")
	if usage.timestamp {
		helpers = append(helpers, "isRFC3339Timestamp")
	}
	if usage.uuid {
		helpers = append(helpers, "isUUID")
	}
	fmt.Fprintf(&builder, "import { %s } from '%s';\n", strings.Join(helpers, ", "), runtimeImportPath)

	for _, definition := range spec.Definitions {
		switch {
		case definition.Schema.Kind == KindString && len(definition.Schema.Enum) > 0:
			emitTypeScriptEnum(&builder, definition)
		case definition.Schema.Kind == KindObject:
			if err := emitTypeScriptInterface(&builder, spec, definition); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("$defs.%s: unsupported TypeScript definition kind %q", definition.Name, definition.Schema.Kind)
		}
	}

	emitTypeScriptUnions(&builder, spec)
	if err := emitTypeScriptGuards(&builder, spec); err != nil {
		return nil, err
	}

	return []byte(builder.String()), nil
}

func emitTypeScriptEnum(builder *strings.Builder, definition *Definition) {
	fmt.Fprintf(builder, "\nexport type %s =", definition.Name)
	for index, value := range definition.Schema.Enum {
		separator := " |"
		if index == len(definition.Schema.Enum)-1 {
			separator = ""
		}
		fmt.Fprintf(builder, " %s%s", singleQuote(value), separator)
	}
	builder.WriteString(";\n")
}

func emitTypeScriptInterface(builder *strings.Builder, spec *Spec, definition *Definition) error {
	fmt.Fprintf(builder, "\nexport interface %s {\n", definition.Name)
	for _, property := range definition.Schema.Properties {
		typeName, err := tsTypeName(spec, property.Schema)
		if err != nil {
			return fmt.Errorf("$defs.%s.%s: %w", definition.Name, property.Name, err)
		}
		if property.Schema.Nullable {
			typeName += " | null"
		}
		optional := "?"
		if property.Required {
			optional = ""
		}
		fmt.Fprintf(builder, "  %s%s: %s;\n", property.Name, optional, typeName)
	}
	builder.WriteString("}\n")
	return nil
}

func tsTypeName(spec *Spec, schema *Schema) (string, error) {
	switch schema.Kind {
	case KindRef:
		if _, ok := spec.Definition(schema.RefName); !ok {
			return "", fmt.Errorf("unknown $ref %q", schema.RefName)
		}
		return schema.RefName, nil
	case KindString:
		if schema.Const != "" {
			return singleQuote(schema.Const), nil
		}
		return "string", nil
	case KindInteger, KindNumber:
		return "number", nil
	case KindBoolean:
		return "boolean", nil
	case KindArray:
		if schema.Items == nil {
			return "", fmt.Errorf("array without items")
		}
		itemType, err := tsTypeName(spec, schema.Items)
		if err != nil {
			return "", err
		}
		if strings.Contains(itemType, " | ") {
			return "(" + itemType + ")[]", nil
		}
		return itemType + "[]", nil
	default:
		return "", fmt.Errorf("unsupported type %q", schema.Kind)
	}
}

func emitTypeScriptUnions(builder *strings.Builder, spec *Spec) {
	emitStringUnion(builder, "ClientCommandType", wireTypes(spec.CommandMessages()))
	emitStringUnion(builder, "ServerEventType", wireTypes(spec.EventMessages()))
	emitStringUnion(builder, "GuardedCommandType", wireTypes(spec.GuardedMessages()))

	emitTypeUnion(builder, "ClientCommand", definitionNames(spec.CommandMessages()))

	emptyGuarded := emptyGuardedMessages(spec)
	emitStringUnion(builder, "EmptyGuardedCommandType", wireTypes(emptyGuarded))
	builder.WriteString("\nexport const EMPTY_GUARDED_COMMANDS: { readonly [K in EmptyGuardedCommandType]: Extract<ClientCommand, { type: K }> } = {\n")
	for _, wireType := range wireTypes(emptyGuarded) {
		fmt.Fprintf(builder, "  %s: { type: %s, payload: {} },\n", singleQuote(wireType), singleQuote(wireType))
	}
	builder.WriteString("};\n")

	emitTypeUnion(builder, "ServerEvent", definitionNames(spec.EventMessages()))

	builder.WriteString("\nexport const GUARDED_COMMAND_TYPES: readonly GuardedCommandType[] = [\n")
	for _, wireType := range wireTypes(spec.GuardedMessages()) {
		fmt.Fprintf(builder, "  %s,\n", singleQuote(wireType))
	}
	builder.WriteString("];\n")

	builder.WriteString("\nexport const SERVER_EVENT_TYPES: readonly ServerEventType[] = [\n")
	for _, wireType := range wireTypes(spec.EventMessages()) {
		fmt.Fprintf(builder, "  %s,\n", singleQuote(wireType))
	}
	builder.WriteString("];\n")

	builder.WriteString("\nexport function isGuardedCommand(type: string): type is GuardedCommandType {\n")
	builder.WriteString("  return GUARDED_COMMAND_TYPES.some((commandType) => commandType === type);\n")
	builder.WriteString("}\n")
}

func emitStringUnion(builder *strings.Builder, name string, values []string) {
	fmt.Fprintf(builder, "\nexport type %s =\n", name)
	if len(values) == 0 {
		builder.WriteString("  never;\n")
		return
	}
	for index, value := range values {
		terminator := ";"
		if index < len(values)-1 {
			terminator = ""
		}
		fmt.Fprintf(builder, "  | %s%s\n", singleQuote(value), terminator)
	}
}

func emitTypeUnion(builder *strings.Builder, name string, values []string) {
	fmt.Fprintf(builder, "\nexport type %s =\n", name)
	if len(values) == 0 {
		builder.WriteString("  never;\n")
		return
	}
	for index, value := range values {
		terminator := ";"
		if index < len(values)-1 {
			terminator = ""
		}
		fmt.Fprintf(builder, "  | %s%s\n", value, terminator)
	}
}

func emitTypeScriptGuards(builder *strings.Builder, spec *Spec) error {
	for _, definition := range spec.Definitions {
		switch {
		case definition.Schema.Kind == KindString && len(definition.Schema.Enum) > 0:
			emitTypeScriptEnumGuard(builder, definition)
		case definition.Schema.Kind == KindObject:
			if err := emitTypeScriptObjectGuard(builder, spec, definition); err != nil {
				return err
			}
		}
	}
	emitTypeScriptDispatcher(builder, "isServerEvent", "ServerEvent", spec.EventMessages())
	emitTypeScriptDispatcher(builder, "isClientCommand", "ClientCommand", spec.CommandMessages())
	return nil
}

func emitTypeScriptEnumGuard(builder *strings.Builder, definition *Definition) {
	guard := "is" + definition.Name
	fmt.Fprintf(builder, "\nexport function %s(value: unknown): value is %s {\n", guard, definition.Name)
	builder.WriteString("  return ")
	for index, value := range definition.Schema.Enum {
		if index > 0 {
			builder.WriteString(" || ")
		}
		fmt.Fprintf(builder, "value === %s", singleQuote(value))
	}
	builder.WriteString(";\n}\n")
}

func emitTypeScriptObjectGuard(builder *strings.Builder, spec *Spec, definition *Definition) error {
	guard := "is" + definition.Name
	fmt.Fprintf(builder, "\nexport function %s(value: unknown): value is %s {\n", guard, definition.Name)
	builder.WriteString("  if (!isRecord(value)) {\n    return false;\n  }\n\n")

	if len(definition.Schema.Properties) == 0 {
		builder.WriteString("  return true;\n}\n")
		return nil
	}

	checks := make([]string, 0, len(definition.Schema.Properties))
	for _, property := range definition.Schema.Properties {
		accessor := "value." + property.Name
		check, err := tsCheck(spec, property.Schema, accessor)
		if err != nil {
			return fmt.Errorf("$defs.%s.%s: %w", definition.Name, property.Name, err)
		}
		if !property.Required {
			check = fmt.Sprintf("(!('%s' in value) || %s)", property.Name, check)
		}
		checks = append(checks, check)
	}

	builder.WriteString("  return (\n")
	for index, check := range checks {
		suffix := " &&"
		if index == len(checks)-1 {
			suffix = ""
		}
		fmt.Fprintf(builder, "    %s%s\n", check, suffix)
	}
	builder.WriteString("  );\n}\n")
	return nil
}

func emitTypeScriptDispatcher(builder *strings.Builder, guard, typeName string, messages []Message) {
	fmt.Fprintf(builder, "\nexport function %s(value: unknown): value is %s {\n", guard, typeName)
	builder.WriteString("  if (!isRecord(value)) {\n    return false;\n  }\n\n")
	builder.WriteString("  switch (value.type) {\n")
	for _, message := range messages {
		fmt.Fprintf(builder, "    case %s:\n      return is%s(value);\n", singleQuote(message.WireType), message.DefName)
	}
	builder.WriteString("    default:\n      return false;\n  }\n}\n")
}

func tsCheck(spec *Spec, schema *Schema, accessor string) (string, error) {
	check, err := tsNonNullCheck(spec, schema, accessor)
	if err != nil {
		return "", err
	}
	if schema.Nullable {
		return fmt.Sprintf("(%s === null || (%s))", accessor, check), nil
	}
	return check, nil
}

func tsNonNullCheck(spec *Spec, schema *Schema, accessor string) (string, error) {
	switch schema.Kind {
	case KindRef:
		return tsRefCheck(spec, schema.RefName, accessor)
	case KindString:
		return tsStringCheck(schema, accessor), nil
	case KindInteger:
		return fmt.Sprintf("isInteger(%s)", accessor), nil
	case KindNumber:
		return fmt.Sprintf("typeof %s === 'number'", accessor), nil
	case KindBoolean:
		return fmt.Sprintf("typeof %s === 'boolean'", accessor), nil
	case KindArray:
		return tsArrayCheck(spec, schema, accessor)
	default:
		return "", fmt.Errorf("unsupported type %q", schema.Kind)
	}
}

func tsRefCheck(spec *Spec, refName, accessor string) (string, error) {
	definition, ok := spec.Definition(refName)
	if !ok {
		return "", fmt.Errorf("unknown $ref %q", refName)
	}
	if tsRefIsValidatable(definition.Schema) {
		return fmt.Sprintf("is%s(%s)", definition.Name, accessor), nil
	}
	return "", fmt.Errorf("$ref %q cannot be validated", refName)
}

func tsRefIsValidatable(schema *Schema) bool {
	if schema.Kind == KindObject {
		return true
	}
	return schema.Kind == KindString && len(schema.Enum) > 0
}

func tsStringCheck(schema *Schema, accessor string) string {
	if schema.Const != "" {
		return fmt.Sprintf("%s === %s", accessor, singleQuote(schema.Const))
	}
	switch schema.Format {
	case "date-time":
		return fmt.Sprintf("isRFC3339Timestamp(%s)", accessor)
	case "uuid":
		return fmt.Sprintf("isUUID(%s)", accessor)
	default:
		return fmt.Sprintf("typeof %s === 'string'", accessor)
	}
}

func tsArrayCheck(spec *Spec, schema *Schema, accessor string) (string, error) {
	if schema.Items == nil {
		return "", fmt.Errorf("array without items")
	}
	itemCheck, err := tsCheck(spec, schema.Items, "item")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Array.isArray(%s) && %s.every((item) => %s)", accessor, accessor, itemCheck), nil
}

type typeScriptUsage struct {
	integer   bool
	timestamp bool
	uuid      bool
}

func collectTypeScriptUsage(spec *Spec) typeScriptUsage {
	usage := typeScriptUsage{}
	visited := make(map[string]bool)
	for _, definition := range spec.Definitions {
		scanTypeScriptUsage(spec, definition.Schema, &usage, visited)
	}
	return usage
}

func scanTypeScriptUsage(spec *Spec, schema *Schema, usage *typeScriptUsage, visited map[string]bool) {
	switch schema.Kind {
	case KindObject:
		for _, property := range schema.Properties {
			scanTypeScriptUsage(spec, property.Schema, usage, visited)
		}
	case KindString:
		switch schema.Format {
		case "date-time":
			usage.timestamp = true
		case "uuid":
			usage.uuid = true
		}
	case KindInteger:
		usage.integer = true
	case KindArray:
		if schema.Items != nil {
			scanTypeScriptUsage(spec, schema.Items, usage, visited)
		}
	case KindRef:
		if visited[schema.RefName] {
			return
		}
		visited[schema.RefName] = true
		if definition, ok := spec.Definition(schema.RefName); ok {
			scanTypeScriptUsage(spec, definition.Schema, usage, visited)
		}
	}
}

// emptyGuardedMessages returns guarded commands whose payload has no required properties.
func emptyGuardedMessages(spec *Spec) []Message {
	var empty []Message
	for _, message := range spec.GuardedMessages() {
		definition, ok := spec.Definition(message.PayloadType)
		if !ok || definition.Schema.Kind != KindObject {
			continue
		}
		if !hasRequiredProperty(definition.Schema) {
			empty = append(empty, message)
		}
	}
	return empty
}

func hasRequiredProperty(schema *Schema) bool {
	for _, property := range schema.Properties {
		if property.Required {
			return true
		}
	}
	return false
}

func wireTypes(messages []Message) []string {
	values := make([]string, 0, len(messages))
	for _, message := range messages {
		values = append(values, message.WireType)
	}
	return values
}

func definitionNames(messages []Message) []string {
	names := make([]string, 0, len(messages))
	for _, message := range messages {
		names = append(names, message.DefName)
	}
	return names
}

func singleQuote(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return "'" + escaped + "'"
}
