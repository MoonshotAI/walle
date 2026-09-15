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

	// A single plain part names the keyword the error is about, so the schema
	// holding it is the root. An indexed part such as "anyOf{1}" names a branch
	// instead, and has to be descended into even when it stands alone.
	if len(parts) == 1 && !isIndexedPathPart(parts[0]) {
		return current, nil
	}

	for _, part := range parts {
		if isIndexedPathPart(part) {
			next, err := resolveIndexedPathPart(current, part)
			if err != nil {
				return nil, invalidPathErr
			}
			current = next
			continue
		}

		next, ok := current[part].(SchemaDict)
		if !ok {
			return nil, invalidPathErr
		}
		current = next
	}

	return current, nil
}

// isIndexedPathPart reports whether a path part addresses a list entry, as
// "anyOf{1}" does.
func isIndexedPathPart(part string) bool {
	return strings.Contains(part, "{") && strings.Contains(part, "}")
}

// resolveIndexedPathPart follows a part such as "anyOf{1}" into the list entry it
// names.
func resolveIndexedPathPart(current Schema, part string) (Schema, error) {
	invalidPathErr := fmt.Errorf("invalid path part: %s", part)

	baseParts := strings.Split(part, "{")
	if len(baseParts) < 2 {
		return nil, invalidPathErr
	}

	index, err := strconv.Atoi(strings.TrimSuffix(baseParts[1], "}"))
	if err != nil {
		return nil, invalidPathErr
	}

	list, ok := current[baseParts[0]].(SchemaList)
	if !ok {
		return nil, invalidPathErr
	}
	if index < 0 || index >= len(list) {
		return nil, invalidPathErr
	}

	entry, ok := list[index].(SchemaDict)
	if !ok {
		return nil, invalidPathErr
	}
	return entry, nil
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

// resolveDictAtPath returns the dict that path points at. extractSubSchema treats a
// single-part path as the root, which is wrong for paths like "properties", so that
// case is resolved here explicitly. An indexed part such as "anyOf{0}" already names
// a dict of its own and is left to extractSubSchema even when it stands alone.
func resolveDictAtPath(schema Schema, path schemaPath) (Schema, error) {
	if len(path.Parts) > 1 || (!path.IsRoot() && isIndexedPathPart(path.Last())) {
		return extractSubSchema(schema, path)
	}
	if path.IsRoot() {
		return schema, nil
	}
	next, ok := schema[path.Last()].(SchemaDict)
	if !ok {
		return nil, fmt.Errorf("invalid path: %s", path.String())
	}
	return next, nil
}

// deletes the given keys from the schema at path
func SimplifyRemoveSchemaKeys(keys []string) SimplifyFunc {
	keysCopy := append([]string(nil), keys...)
	return func(schema Schema, path schemaPath) Schema {
		current, err := extractSubSchema(schema, path)
		if err != nil {
			return make(Schema)
		}
		for _, k := range keysCopy {
			delete(current, k)
		}
		return schema
	}
}

// simplifyRemoveKeysAtNode deletes keys from the object that path names.
// A single-part path such as "items" is that nested object, not the root.
func simplifyRemoveKeysAtNode(keys ...string) SimplifyFunc {
	keysCopy := append([]string(nil), keys...)
	return func(schema Schema, path schemaPath) Schema {
		current, err := resolveDictAtPath(schema, path)
		if err != nil {
			return make(Schema)
		}
		for _, k := range keysCopy {
			delete(current, k)
		}
		return schema
	}
}

// simplifyFuncForAnyOfParentConflicts picks how to resolve a keyword that a node
// states both directly and inside its anyOf branches. A real constraint is pushed
// into the branches so that the stricter of the two values survives; an annotation
// carries no constraint to preserve, so the outer copy is simply dropped.
func simplifyFuncForAnyOfParentConflicts(conflicts []string) SimplifyFunc {
	for _, keyword := range conflicts {
		if !CommonKeywords[keyword] {
			return SimplifyDistributeAnyOfParent
		}
	}
	return SimplifyRemoveSchemaKeys(conflicts)
}

// Pure description/title conflicts remove only those keys at the error path (outer layer).
func simplifyFuncForKeywordConflicts(conflicts []string) SimplifyFunc {
	commonKeys := make([]string, 0, len(conflicts))
	for _, k := range conflicts {
		if !CommonKeywords[k] {
			return SimplifyRemoveParentSchema
		}
		commonKeys = append(commonKeys, k)
	}
	if len(commonKeys) == 0 {
		return SimplifyRemoveParentSchema
	}
	return SimplifyRemoveSchemaKeys(commonKeys)
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

// SimplifyPruneRequired drops the entries of required that name no declared
// property. Demanding a property the schema says nothing about is legal but
// leaves the enforcer nothing to generate, and removing the whole keyword would
// also release the properties that are declared and genuinely required.
func SimplifyPruneRequired(schema Schema, path schemaPath) Schema {
	holder, err := resolveDictAtPath(schema, path.Parent())
	if err != nil {
		return make(Schema)
	}

	required, ok := holder[Required].(SchemaList)
	if !ok {
		delete(holder, Required)
		return schema
	}

	props, _ := holder[Properties].(SchemaDict)
	kept := make(SchemaList, 0, len(required))
	for _, entry := range required {
		name, ok := entry.(string)
		if !ok {
			continue
		}
		if _, declared := props[name]; declared {
			kept = append(kept, name)
		}
	}

	if len(kept) == 0 {
		delete(holder, Required)
		return schema
	}
	holder[Required] = kept
	return schema
}

// simplifyIntersectEnumWithType keeps only the enum values the declared type
// admits. Those are the ones an instance could ever hold; the rest are already
// unreachable, so dropping them changes nothing about what the schema accepts.
//
// No value surviving means the two keywords leave nothing to satisfy them, and one
// of them has to give. The enum goes and the type stays, because that is the
// tightest schema still expressible here: keeping the enum instead would accept
// values of a type the schema ruled out.
func (v *keywordValidators) simplifyIntersectEnumWithType(typeList []string) SimplifyFunc {
	return func(schema Schema, path schemaPath) Schema {
		holder, err := resolveDictAtPath(schema, path.Parent())
		if err != nil {
			return make(Schema)
		}

		enum, ok := holder[Enum].(SchemaList)
		if !ok {
			delete(holder, Enum)
			return schema
		}

		kept := make(SchemaList, 0, len(enum))
		for _, val := range enum {
			for _, t := range typeList {
				if matches, err := v.utils.IsTypeMatch(val, t, nil, path); err == nil && matches {
					kept = append(kept, val)
					break
				}
			}
		}

		if len(kept) == 0 {
			delete(holder, Enum)
			return schema
		}
		holder[Enum] = kept
		return schema
	}
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

// SimplifyDegradeEnclosingSchema empties the schema that holds the keyword at
// path. Used when a keyword combination is unsatisfiable: dropping the individual
// bounds would turn "impossible" into "anything goes", which is a far bigger
// change in meaning than leaving the field unconstrained on purpose.
func SimplifyDegradeEnclosingSchema(schema Schema, path schemaPath) Schema {
	return SimplifyRemoveParentSchema(schema, path.Parent())
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

func SimplifyRemoveTitle(schema Schema, path schemaPath) Schema {
	err := removeAtPath(schema, path.Parent(), Title, true)
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

// SimplifyDistributeAnyOfParent pushes the constraints sitting beside anyOf into
// each of its branches, where the enforcer can act on them, and drops the branches
// that cannot agree with them. Deleting the parent constraints instead would let
// the node accept what they ruled out.
//
// A node that cannot be rewritten this way -- a malformed anyOf, for instance --
// is emptied. Returning it untouched would leave the retry loop reporting the same
// error until it gives up and throws away the whole document.
func SimplifyDistributeAnyOfParent(schema Schema, path schemaPath) Schema {
	node, err := resolveDictAtPath(schema, path)
	if err != nil {
		return make(Schema)
	}

	if _, _, ok := distributableAnyOf(SchemaDict(node), SchemaDict(schema)); !ok {
		for key := range node {
			delete(node, key)
		}
		return schema
	}

	distributeAnyOf(SchemaDict(node), SchemaDict(schema))
	return schema
}

// SimplifyDropContradictingRefSibling replaces the node at path with the merge of
// its own constraints and the schema it references, minus the keyword the two
// sides disagree on. Keeping the rest is the point: a node whose enum cannot
// overlap the definition's usually still agrees on the type, and emptying the
// whole node would throw that agreement away too. When the reference cannot be
// inlined at all, a recursive definition for instance, there is nothing to keep
// and the node is emptied.
func SimplifyDropContradictingRefSibling(schema Schema, path schemaPath) Schema {
	node, err := extractSubSchema(schema, path)
	if err != nil {
		return make(Schema)
	}

	merged, ok := mergeRefSiblingDroppingContradiction(SchemaDict(schema), SchemaDict(node))
	if !ok {
		return SimplifyRemoveParentSchema(schema, path)
	}

	for key := range node {
		delete(node, key)
	}
	for key, value := range merged {
		node[key] = value
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
