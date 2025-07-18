package walle

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type SimplifyFunc func(schema Schema, path schemaPath) Schema

func extractSubSchema(schema Schema, path schemaPath) (Schema, error) {
	current := schema
	parts := path.Parts

	invalidPathErr := fmt.Errorf("invalid path: %s", path.String())
	if len(parts) > 1 {
		for _, part := range parts {
			// Handle anyOf{index} pattern
			if strings.Contains(part, "{") && strings.Contains(part, "}") {
				baseParts := strings.Split(part, "{")
				if len(baseParts) < 2 {
					return nil, invalidPathErr
				}
				base := baseParts[0]
				schemaIndex, err := strconv.Atoi(strings.TrimSuffix(baseParts[1], "}"))
				if err != nil {
					return nil, invalidPathErr
				}
				baseValue, exists := current[base]
				if !exists {
					return nil, invalidPathErr
				}

				currentList, ok := baseValue.(SchemaList)
				if !ok {
					return nil, invalidPathErr
				}

				if schemaIndex < 0 || schemaIndex >= len(currentList) {
					return nil, invalidPathErr
				}

				itemDict, ok := currentList[schemaIndex].(SchemaDict)
				if !ok {
					return nil, invalidPathErr
				}

				current = itemDict

			} else {
				if next, ok := current[part].(SchemaDict); ok {
					current = next
				} else {
					return nil, fmt.Errorf("invalid path: %s", path.String())
				}
			}
		}
	}

	return current, nil
}

func removeAtPath(schema Schema, path schemaPath, targetKey string, mustExist bool) error {
	current, err := extractSubSchema(schema, path)
	if err != nil {
		return err
	}

	if _, exists := current[targetKey]; !exists && mustExist {
		return fmt.Errorf("key '%s' not found at path: %s", targetKey, path.String())
	}

	delete(current, targetKey)
	return nil
}

func removeDuplicateArrayItem(schema Schema, path schemaPath) error {
	current := schema
	parts := path.Parts

	if len(parts) > 1 {
		for i := 0; i < len(parts)-1; i++ {
			part := parts[i]
			if next, ok := current[part].(SchemaDict); ok {
				current = next
			} else {
				return fmt.Errorf("invalid path: %s", path.String())
			}
		}
	}

	// backup
	targetKey := parts[len(parts)-1]

	// use map to remove duplicate items
	if array, ok := current[targetKey].(SchemaList); ok {
		seen := make(map[string]bool)
		uniqueItems := make(SchemaList, 0)

		for _, item := range array {
			if itemStr, ok := item.(string); ok {
				if !seen[itemStr] {
					seen[itemStr] = true
					uniqueItems = append(uniqueItems, item)
				}
			}
		}

		// write back
		current[targetKey] = uniqueItems
	} else {
		return fmt.Errorf("path '%s' does not point to an array", path.String())
	}

	return nil
}

func SimplifyDefault(schema Schema, _ schemaPath) Schema {
	return schema
}

func SimplifyRemoveProperties(schema Schema, path schemaPath) Schema {
	// remove properties
	err := removeAtPath(schema, path.Parent(), Properties, true)
	if err != nil {
		return make(Schema)
	}

	// remove required
	err = removeAtPath(schema, path.Parent(), Required, false)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveRequired(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), Required, true)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveEnum(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), Enum, true)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveAdditionalProperties(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), AdditionalProperties, true)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveRef(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), Ref, true)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveDefs(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), Defs, true)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveID(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), Id, true)
	if err != nil {
		return make(Schema)
	}
	return schema
}

func SimplifyRemovePattern(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), Pattern, true)
	if err != nil {
		return make(Schema)
	}
	return schema
}

func SimplifyRemoveConstraints(schema Schema, path schemaPath) Schema {
	constraints := []string{MinLength, MaxLength, Minimum, Maximum, MinItems, MaxItems}

	for _, constraint := range constraints {
		err := removeAtPath(schema, path.Parent(), constraint, false)
		if err != nil {
			return make(Schema)
		}
	}

	return schema
}

func SimplifyRemoveType(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path, Type, true)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveItems(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), Items, true)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveDescription(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), Description, true)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveAnyOf(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), AnyOf, true)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveDuplicateType(schema Schema, path schemaPath) Schema {
	err := removeDuplicateArrayItem(schema, path)
	if err != nil {
		return make(Schema)
	}

	return schema
}

func SimplifyRemoveParentSchema(schema Schema, path schemaPath) Schema {
	current, err := extractSubSchema(schema, path)
	if err != nil {
		return make(Schema)
	}

	for key := range current {
		delete(current, key)
	}

	return schema
}

func SimplifyRemoveSubSchema(schema Schema, path schemaPath) Schema {
	current := schema
	parts := path.Parts
	targetKey := parts[len(parts)-1]

	if len(parts) > 1 {
		current, err := extractSubSchema(current, path)
		if err != nil {
			return make(Schema)
		}
		for key := range current {
			delete(current, key)
		}
	} else {
		if _, ok := current[targetKey].(SchemaDict); ok {
			lastMap := current[targetKey].(SchemaDict)
			for key := range lastMap {
				delete(lastMap, key)
			}
		} else {
			return make(Schema)
		}
	}

	return schema
}

func SimplifyRemoveDefsEmptySubSchema(schema Schema, path schemaPath) Schema {
	current := schema

	if _, ok := current[Defs]; ok {
		for keyName := range current[Defs].(SchemaDict) {
			if len(keyName) == 0 {
				delete(current[Defs].(SchemaDict), keyName)
			} else if strings.Contains(keyName, "/") {
				delete(current[Defs].(SchemaDict), keyName)
			}
		}
	}

	return schema
}

func SimplifyNegativeVal(schema Schema, path schemaPath) Schema {
	current, err := extractSubSchema(schema, path)
	if err != nil {
		return make(Schema)
	}

	if minLength, ok := current[MinLength]; ok {
		if val, ok := minLength.(float64); ok && val < 0 {
			current[MinLength] = 0.0
		}
	}

	if maxLength, ok := current[MaxLength]; ok {
		if val, ok := maxLength.(float64); ok && val < 0 {
			current[MaxLength] = float64(math.MaxInt64)
		}
	}

	if minItems, ok := current[MinItems]; ok {
		if val, ok := minItems.(float64); ok && val < 0 {
			current[MinItems] = 0.0
		}
	}

	if maxItems, ok := current[MaxItems]; ok {
		if val, ok := maxItems.(float64); ok && val < 0 {
			current[MaxItems] = float64(math.MaxInt64)
		}
	}

	return schema
}
