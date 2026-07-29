package walle

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
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

func refToSchemaPath(ref string) (schemaPath, error) {
	if ref == "#" {
		return rootSchemaPath, nil
	}
	if !strings.HasPrefix(ref, "#") {
		return schemaPath{}, fmt.Errorf("invalid ref: %s", ref)
	}
	parts := strings.Split(ref[2:], "/")
	if len(parts) == 0 {
		return rootSchemaPath, nil
	}
	return newSchemaPathFromParts(parts), nil
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

// hoistedRefPrefix is the synthetic $defs key prefix used for subschemas that
// were hoisted out of a non-$defs location.
const hoistedRefPrefix = "__ref_"

// hoistLocalRefs rewrites local JSON pointers that do not target $defs, such as
// "#/properties/foo", into root level $defs entries, so that the resulting
// schema only ever contains "#" and "#/$defs/<name>" references.
//
// This keeps the MFJS reference contract ("#/$defs/" only) as an invariant of
// the schema handed to the enforcer, while letting callers write the plain
// JSON Schema pointers that draft 2020-12 allows.
//
// It is best effort and never reports errors: a pointer that cannot be resolved
// to a subschema is left untouched, so the validator still rejects it through
// the existing rules. The input schema is never mutated; a copy is returned only
// when something was actually hoisted.
func hoistLocalRefs(schema SchemaDict) SchemaDict {
	if len(schema) == 0 || !hasHoistableRef(schema) {
		return schema
	}

	root, ok := deepCopyValue(schema).(SchemaDict)
	if !ok {
		return schema
	}
	defs, ok := rootDefs(root)
	if !ok {
		return schema
	}

	keyByVariant := make(map[string]string)
	// Newly hoisted copies are queued instead of being reached by recursion, so
	// that $defs is never walked while it is being extended.
	queue := []any{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		rewriteRefs(node, root, defs, keyByVariant, &queue)
	}

	if len(keyByVariant) == 0 {
		return schema
	}
	root[Defs] = defs
	return root
}

// hasHoistableRef reports whether the subtree contains a local pointer that
// needs to be hoisted, so that the common case costs one read-only walk.
func hasHoistableRef(node any) bool {
	switch n := node.(type) {
	case SchemaDict:
		if ref, ok := n[Ref].(string); ok && isHoistableRef(ref) {
			return true
		}
		for _, value := range n {
			if hasHoistableRef(value) {
				return true
			}
		}
	case SchemaList:
		for _, item := range n {
			if hasHoistableRef(item) {
				return true
			}
		}
	}
	return false
}

// isHoistableRef matches local pointers other than "#" and "#/$defs/...", both
// of which already resolve on the enforcer side.
func isHoistableRef(ref string) bool {
	return strings.HasPrefix(ref, "#/") && !strings.HasPrefix(ref, "#/"+Defs+"/")
}

// rewriteRefs walks by value shape rather than by keyword, so pointers nested in
// anyOf branches, items, or any future keyword are rewritten without the walk
// needing to know about them.
func rewriteRefs(node any, root, defs SchemaDict, keyByVariant map[string]string, queue *[]any) {
	switch n := node.(type) {
	case SchemaDict:
		if ref, ok := n[Ref].(string); ok && isHoistableRef(ref) {
			if name, hoisted := hoistRef(ref, n, root, defs, keyByVariant, queue); hoisted {
				n[Ref] = "#/" + Defs + "/" + name
			}
		}
		// Iterate over a snapshot: hoisting may add keys to $defs, which is a
		// child of root and would otherwise be mutated during its own range.
		for _, key := range mapKeys(n) {
			if key != Ref {
				rewriteRefs(n[key], root, defs, keyByVariant, queue)
			}
		}
	case SchemaList:
		for _, item := range n {
			rewriteRefs(item, root, defs, keyByVariant, queue)
		}
	}
}

func hoistRef(ref string, site, root, defs SchemaDict, keyByVariant map[string]string, queue *[]any) (string, bool) {
	pointer := strings.TrimPrefix(ref[1:], "/")
	if pointer == "" {
		return "", false
	}

	shadowed := shadowedAnnotations(site)
	variant := pointer + "\x00" + strings.Join(shadowed, ",")
	if name, seen := keyByVariant[variant]; seen {
		return name, true
	}

	target, ok := resolvePointer(root, pointer)
	if !ok {
		return "", false
	}
	copied, ok := deepCopyValue(target).(SchemaDict)
	if !ok {
		return "", false
	}
	for _, key := range shadowed {
		delete(copied, key)
	}

	name := uniqueDefName(pointer, defs)
	defs[name] = copied
	keyByVariant[variant] = name
	*queue = append(*queue, copied)
	return name, true
}

// shadowedAnnotations lists the annotation keywords the referencing site
// defines itself, in a stable order so that variants dedupe reliably.
func shadowedAnnotations(site SchemaDict) []string {
	var keys []string
	for key := range site {
		if CommonKeywords[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// resolvePointer walks an RFC 6901 pointer (without the leading "#/") and
// requires the target to be a subschema.
func resolvePointer(root SchemaDict, pointer string) (SchemaDict, bool) {
	var current any = root
	for _, token := range strings.Split(pointer, "/") {
		token = unescapePointerToken(token)
		switch node := current.(type) {
		case SchemaDict:
			value, exists := node[token]
			if !exists {
				return nil, false
			}
			current = value
		case SchemaList:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(node) {
				return nil, false
			}
			current = node[index]
		default:
			return nil, false
		}
	}
	target, ok := current.(SchemaDict)
	return target, ok
}

// unescapePointerToken applies the RFC 6901 escapes; "~1" must be decoded
// before "~0" so that "~01" stays a literal "~1".
func unescapePointerToken(token string) string {
	token = strings.ReplaceAll(token, "~1", "/")
	return strings.ReplaceAll(token, "~0", "~")
}

// uniqueDefName derives a readable definition name from the pointer so that the
// origin of a hoisted subschema stays visible in logs and error messages.
func uniqueDefName(pointer string, defs SchemaDict) string {
	var sanitized strings.Builder
	sanitized.WriteString(hoistedRefPrefix)
	for _, r := range pointer {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			sanitized.WriteRune(r)
		default:
			sanitized.WriteRune('_')
		}
	}
	base := sanitized.String()

	name := base
	for i := 2; ; i++ {
		if _, taken := defs[name]; !taken {
			return name
		}
		name = base + "_" + strconv.Itoa(i)
	}
}

// rootDefs returns the root $defs map, creating an empty one when absent. It
// fails when $defs exists but is not an object, leaving that to the validator.
func rootDefs(root SchemaDict) (SchemaDict, bool) {
	existing, present := root[Defs]
	if !present {
		return make(SchemaDict), true
	}
	defs, ok := existing.(SchemaDict)
	return defs, ok
}

func mapKeys(node SchemaDict) []string {
	keys := make([]string, 0, len(node))
	for key := range node {
		keys = append(keys, key)
	}
	return keys
}

func deepCopyValue(node any) any {
	switch n := node.(type) {
	case SchemaDict:
		copied := make(SchemaDict, len(n))
		for key, value := range n {
			copied[key] = deepCopyValue(value)
		}
		return copied
	case SchemaList:
		copied := make(SchemaList, len(n))
		for i, item := range n {
			copied[i] = deepCopyValue(item)
		}
		return copied
	default:
		return node
	}
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
