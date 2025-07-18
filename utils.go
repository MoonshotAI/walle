package walle

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type schemaPath struct {
	Parts []string
}

var rootSchemaPath = schemaPath{Parts: []string{Root}}

func newSchemaPath(path string) schemaPath {
	if path == "" {
		return schemaPath{Parts: []string{Root}}
	}

	if strings.Contains(path, ".") {
		return schemaPath{Parts: strings.Split(path, ".")}
	}

	return schemaPath{Parts: []string{path}}
}

func newSchemaPathFromParts(parts []string) schemaPath {
	if len(parts) == 0 {
		return schemaPath{Parts: []string{Root}}
	}
	return schemaPath{Parts: parts}
}

func (p schemaPath) Parent() schemaPath {
	if len(p.Parts) <= 1 {
		return schemaPath{Parts: []string{Root}}
	}
	return schemaPath{Parts: p.Parts[:len(p.Parts)-1]}
}

func (p schemaPath) Last() string {
	if len(p.Parts) == 0 {
		return ""
	}
	return p.Parts[len(p.Parts)-1]
}

func (p schemaPath) String() string {
	return strings.Join(p.Parts, ".")
}

func (p schemaPath) IsRoot() bool {
	return (len(p.Parts) == 1 && p.Parts[0] == Root) || len(p.Parts) == 0
}

func (p schemaPath) Append(parts ...string) schemaPath {
	if p.IsRoot() && len(parts) > 0 {
		return schemaPath{Parts: parts}
	}

	newParts := make([]string, len(p.Parts)+len(parts))
	copy(newParts, p.Parts)
	copy(newParts[len(p.Parts):], parts)
	return schemaPath{Parts: newParts}
}

func (p schemaPath) ModifyAnyOfPart(index int) schemaPath {
	if len(p.Parts) == 0 {
		return p
	}

	last := p.Parts[len(p.Parts)-1]
	newParts := make([]string, len(p.Parts))
	copy(newParts, p.Parts)
	newParts[len(newParts)-1] = fmt.Sprintf("%s{%d}", last, index)
	return schemaPath{Parts: newParts}
}

func (p schemaPath) StringWithoutLast() schemaPath {
	if len(p.Parts) <= 1 {
		return schemaPath{Parts: []string{Root}}
	}
	return newSchemaPathFromParts(p.Parts[:len(p.Parts)-1])
}

// validateUtils provides utility functions for schema validation
type validateUtils struct{}

// CalculateSchemaSize calculates the size of a schema
func (u *validateUtils) CalculateSchemaSize(schema SchemaDict) int {
	bytes, err := json.Marshal(schema)
	if err != nil {
		return 0
	}
	return len(bytes)
}

// GetRefPathParts parses $ref path
func (u *validateUtils) GetRefPathParts(ref string, context *validationContext, path schemaPath) ([]string, error) {
	if ref == "#" {
		return []string{Root}, nil
	}
	if !strings.HasPrefix(ref, "#/$defs/") {
		return nil, context.RaiseErrorWithSimplify("only local references are supported", path.Append(Ref), SimplifyRemoveRef)
	}
	return strings.Split(ref[2:], "/"), nil
}

// ResolveRef resolves reference to actual schema
func (u *validateUtils) ResolveRef(root SchemaDict, ref string, context *validationContext, path schemaPath) (SchemaDict, error) {
	if ref == "#" {
		return context.SchemaRoot, nil
	}

	current := root
	parts, err := u.GetRefPathParts(ref, context, path)
	if err != nil {
		return nil, err
	}

	for _, part := range parts {
		if val, ok := current[part]; ok {
			if m, ok := val.(SchemaDict); ok {
				current = m
			} else {
				return nil, context.RaiseError(fmt.Sprintf("invalid $ref path: %s", ref), path)
			}
		} else {
			return nil, context.RaiseError(fmt.Sprintf("invalid $ref path: %s", ref), path)
		}
	}

	return current, nil
}

func (u *validateUtils) ResolveSubschema(root SchemaDict, resolvePath schemaPath, context *validationContext, path schemaPath) (SchemaDict, error) {
	// If path starts with "#/$defs/", use ResolveRef
	if strings.HasPrefix(resolvePath.String(), "#/$defs/") {
		return u.ResolveRef(root, resolvePath.String(), context, path)
	}

	current := root
	parts := resolvePath.Parts

	for _, part := range parts {
		// Handle anyOf{index} pattern
		if strings.Contains(part, "{") && strings.Contains(part, "}") {
			baseParts := strings.Split(part, "{")
			if len(baseParts) < 2 {
				return nil, context.RaiseError(fmt.Sprintf("internal error: invalid format in schemaPath: %s", part), path)
			}
			base := baseParts[0]
			schemaIndex, err := strconv.Atoi(strings.TrimSuffix(baseParts[1], "}"))
			if err != nil {
				return nil, context.RaiseError(fmt.Sprintf("internal error: invalid format in schemaPath: %s", part), path)
			}

			baseValue, exists := current[base]
			if !exists {
				return nil, nil
			}

			currentList, ok := baseValue.(SchemaList)
			if !ok {
				return nil, nil
			}

			if schemaIndex < 0 || schemaIndex >= len(currentList) {
				return nil, nil
			}

			itemDict, ok := currentList[schemaIndex].(SchemaDict)
			if !ok {
				return nil, nil
			}

			current = itemDict
		} else {
			// Regular property access
			nextValue, exists := current[part]
			if !exists {
				return nil, nil
			}

			nextDict, ok := nextValue.(SchemaDict)
			if !ok {
				return nil, nil
			}
			current = nextDict
		}
	}

	return current, nil
}

func (u *validateUtils) IsTypeMatch(value any, expectedType string, context *validationContext, path schemaPath) (bool, error) {
	switch expectedType {
	case String:
		_, ok := value.(string)
		return ok, nil
	case Number:
		switch val := value.(type) {
		case float64:
			// assert json.Unmarshal get float64
			if err := u.IsValidNumber(val, context, path); err != nil {
				return false, err
			}
			return true, nil
		default:
			return false, context.RaiseErrorWithSimplify("not a valid number", path, SimplifyDefault)
		}
	case Integer:
		switch val := value.(type) {
		case float64:
			if err := u.IsValidInteger(val, context, path); err != nil {
				return false, err
			}
			return true, nil
		default:
			return false, context.RaiseErrorWithSimplify("not a valid integer", path, SimplifyDefault)
		}
	case Boolean:
		_, ok := value.(bool)
		return ok, nil
	case Null:
		return value == nil, nil
	case Array:
		_, ok := value.(SchemaList)
		return ok, nil
	case Object:
		_, ok := value.(SchemaDict)
		return ok, nil
	default:
		return false, context.RaiseErrorWithSimplify("invalid type", path, SimplifyRemoveParentSchema)
	}
}

func (u *validateUtils) IsValidInteger(value float64, context *validationContext, path schemaPath) error {
	if math.Floor(value) != value {
		return context.RaiseErrorWithSimplify("not a valid integer", path, SimplifyDefault)
	}
	return nil
}

func (u *validateUtils) IsValidNumber(value float64, context *validationContext, path schemaPath) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return context.RaiseErrorWithSimplify("invalid number: NaN or Infinity not allowed", path, SimplifyDefault)
	}

	// not support scientific notation
	strVal := fmt.Sprintf("%f", value)
	if strings.Contains(strings.ToLower(strVal), "e") {
		return context.RaiseErrorWithSimplify("invalid number format: scientific notation not allowed", path, SimplifyDefault)
	}

	// Check leading zeros
	if strings.HasPrefix(strVal, "0") || strings.HasPrefix(strVal, "-0") {
		if len(strVal) == 1 || (strings.HasPrefix(strVal, "-") && len(strVal) == 2) {
			return nil // 0 or -0 is valid
		}
		nextCharPos := 1
		if strings.HasPrefix(strVal, "-") {
			nextCharPos = 2
		}
		if strVal[nextCharPos] != '.' {
			return context.RaiseErrorWithSimplify("invalid number format: leading zero not allowed for integers", path, SimplifyDefault)
		}
	}
	return nil
}
