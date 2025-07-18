package walle

import (
	"fmt"
	"slices"
	"strings"
)

// keywordValidators provides validation functions for different keywords
type keywordValidators struct {
	utils  *validateUtils
	config *SchemaValidatorConfig
}

func newKeywordValidators(config *SchemaValidatorConfig) *keywordValidators {
	return &keywordValidators{
		utils:  &validateUtils{},
		config: config,
	}
}

// some type validation is done in validateTypeAndKeywords
func (v *keywordValidators) ValidateType(value any, context *validationContext, path schemaPath) error {
	switch val := value.(type) {
	case string:
		return nil
	case SchemaList:
		if v.config.IsStrict() || v.config.IsTest() {
			// Check for duplicates
			seen := make(map[string]struct{})
			for _, t := range val {
				if _, exists := seen[t.(string)]; exists {
					return context.RaiseErrorWithSimplify("duplicate types in type array", path, SimplifyRemoveDuplicateType)
				}
				seen[t.(string)] = struct{}{}
			}
		}
		return nil
	default:
		return context.RaiseErrorWithSimplify("type must be string or array of strings", path.Parent(), SimplifyRemoveParentSchema)
	}
}

func (v *keywordValidators) ValidateProperties(value any, context *validationContext, path schemaPath) error {
	props, ok := value.(SchemaDict)
	if !ok {
		return context.RaiseErrorWithSimplify("properties must be an object", path, SimplifyRemoveProperties)
	}

	for propName, propSchema := range props {
		if v.config.IsStrict() || v.config.IsTest() {
			if propName == "" {
				return context.RaiseErrorWithSimplify("property name cannot be empty", path, SimplifyDefault)
			}
		}

		if InvalidPropertyNames[propName] {
			return context.RaiseErrorWithSimplify(fmt.Sprintf("property name '%s' is reserved for JSON Schema keywords", propName), path, SimplifyRemoveSubSchema)
		}

		if _, ok := propSchema.(SchemaDict); !ok {
			return context.RaiseErrorWithSimplify(fmt.Sprintf("property schema for '%s' must be an object", propName), path, SimplifyRemoveSubSchema)
		}
	}

	return nil
}

func (v *keywordValidators) ValidateRequired(value any, context *validationContext, path schemaPath) error {
	required, ok := value.(SchemaList)
	if !ok {
		return context.RaiseErrorWithSimplify("required must be an array", path, SimplifyRemoveRequired)
	}

	// Empty array is valid
	if len(required) == 0 {
		return nil
	}

	// Check that all required properties are strings
	for _, prop := range required {
		propStr, ok := prop.(string)
		if !ok {
			return context.RaiseErrorWithSimplify("items in required array must be strings", path, SimplifyRemoveRequired)
		}

		if v.config.IsStrict() || v.config.IsTest() {
			if propStr == "" {
				return context.RaiseErrorWithSimplify("property names in required array cannot be empty", path, SimplifyRemoveRequired)
			}
		}
	}

	// Check for duplicates
	if v.config.IsStrict() || v.config.IsTest() {
		seen := make(map[string]struct{})
		for _, prop := range required {
			propStr := prop.(string)
			if _, exists := seen[propStr]; exists {
				return context.RaiseErrorWithSimplify(fmt.Sprintf("duplicate items in required array: %s", propStr), path, SimplifyRemoveDuplicateType)
			}
			seen[propStr] = struct{}{}
		}
	}

	// Get current object's properties
	parentPath := path.Parent()

	var current SchemaDict
	if parentPath.IsRoot() {
		current = context.SchemaRoot
	} else {
		var err error
		current, err = v.utils.ResolveSubschema(context.SchemaRoot, parentPath, context, path)
		if err != nil || current == nil {
			return context.RaiseError(fmt.Sprintf("invalid path: %s", parentPath), path)
		}
	}

	props, hasProps := current[Properties].(SchemaDict)
	if v.config.IsStrict() || v.config.IsTest() {
		// Check if current schema is object type
		schemaType, hasType := current[Type]
		if !hasType {
			return context.RaiseErrorWithSimplify("required keyword must be used with object", path, SimplifyDefault)
		}

		switch t := schemaType.(type) {
		case string:
			if t != Object {
				return context.RaiseErrorWithSimplify("required keyword must be used with object", path, SimplifyDefault)
			}
		case SchemaList:
			if len(t) != 1 || t[0] != Object {
				return context.RaiseErrorWithSimplify("required keyword must be used with object", path, SimplifyDefault)
			}
		}

		// Check if properties exists
		if !hasProps {
			return context.RaiseErrorWithSimplify("required specified but 'properties' keyword is missing", path, SimplifyDefault)
		}
	}

	// Check if all required fields are defined in properties
	for _, prop := range required {
		propStr := prop.(string)
		if _, exists := props[propStr]; !exists {
			return context.RaiseErrorWithSimplify(fmt.Sprintf("required property '%s' is not defined in properties", propStr), path, SimplifyRemoveRequired)
		}
	}

	return nil
}

func (v *keywordValidators) ValidateEnum(value any, context *validationContext, path schemaPath) error {
	enum, ok := value.(SchemaList)
	if !ok {
		return context.RaiseErrorWithSimplify("enum must be an array", path, SimplifyRemoveEnum)
	}

	if len(enum) == 0 {
		return context.RaiseErrorWithSimplify("enum array cannot be empty", path, SimplifyRemoveEnum)
	}

	if len(enum) > v.config.MaxEnumItems {
		return context.RaiseErrorWithSimplify(
			fmt.Sprintf("enum array cannot have more than %d items", v.config.MaxEnumItems),
			path, SimplifyRemoveEnum,
		)
	}

	// Get parent schema's type
	var parentSchema SchemaDict
	if path.String() == Enum { // root level
		parentSchema = context.SchemaRoot
	} else {
		parentPath := path.Parent()

		var err error
		parentSchema, err = v.utils.ResolveSubschema(context.SchemaRoot, parentPath, context, path)
		if err != nil || parentSchema == nil {
			return context.RaiseError(fmt.Sprintf("invalid path: %s", parentPath), path)
		}
	}

	currentType, hasType := parentSchema[Type]
	if !hasType {
		return context.RaiseErrorWithSimplify("type is not defined", path.Parent(), SimplifyRemoveParentSchema)
	}

	// Convert type to list for consistent handling
	var typeList []string
	switch t := currentType.(type) {
	case string:
		typeList = []string{t}
	case SchemaList:
		typeList = make([]string, 0, len(t))
		for _, item := range t {
			typeStr, ok := item.(string)
			if !ok {
				return context.RaiseErrorWithSimplify(
					"type array must be an array of strings",
					path.Parent(), SimplifyRemoveParentSchema,
				)
			}
			typeList = append(typeList, typeStr)
		}
	default:
		return context.RaiseErrorWithSimplify("invalid type value", path.Parent(), SimplifyRemoveParentSchema)
	}

	// Strict constraint for type arrays with enum
	if len(typeList) > 2 || (len(typeList) == 2 && !slices.Contains(typeList, Null)) {
		return context.RaiseErrorWithSimplify(
			"when type is array and enum is specified, currently only 2 types are supported, "+
				"one must be null, and the other can be string/integer/number/boolean",
			path.Parent(), SimplifyRemoveParentSchema,
		)
	} else if len(typeList) == 2 {
		var otherType string
		for _, t := range typeList {
			if t != Null {
				otherType = t
				break
			}
		}

		if otherType != String && otherType != Integer && otherType != Number && otherType != Boolean {
			return context.RaiseErrorWithSimplify(
				fmt.Sprintf("invalid type combination with enum: %s + null", otherType),
				path.Parent(), SimplifyRemoveParentSchema,
			)
		}
	}

	// Validate each enum value matches at least one type in typeList
	for _, val := range enum {
		matchesAnyType := false
		for _, t := range typeList {
			matches, err := v.utils.IsTypeMatch(val, t, context, path)
			if err != nil {
				return err
			}
			if matches {
				matchesAnyType = true
				break
			}
		}

		if !matchesAnyType {
			return context.RaiseErrorWithSimplify(
				fmt.Sprintf("enum value (%v) does not match any type in %v", val, typeList),
				path, SimplifyRemoveEnum,
			)
		}
	}

	// Check string length constraint for string enums
	limitLengthTypes := []string{String, Number, Integer}
	hasLimitType := false
	for _, t := range typeList {
		for _, lt := range limitLengthTypes {
			if t == lt {
				hasLimitType = true
				break
			}
		}
		if hasLimitType {
			break
		}
	}

	if hasLimitType && len(enum) > v.config.MaxEnumStringCheckThreshold {
		totalLength := 0
		for _, val := range enum {
			switch v := val.(type) {
			case string:
				totalLength += len(v)
			case float64:
				// Convert numbers to string and count their length
				totalLength += len(fmt.Sprintf("%v", v))
			}
		}

		if totalLength > v.config.MaxEnumStringLength {
			return context.RaiseErrorWithSimplify(
				fmt.Sprintf("total string length of enum values (%d) exceeds maximum limit of %d "+
					"characters when enum has more than %d values",
					totalLength, v.config.MaxEnumStringLength, v.config.MaxEnumStringCheckThreshold),
				path, SimplifyRemoveEnum,
			)
		}
	}

	return nil
}

func (v *keywordValidators) ValidateItems(value any, context *validationContext, path schemaPath) error {
	_, ok := value.(SchemaDict)
	if !ok {
		return context.RaiseErrorWithSimplify("items must be an object", path, SimplifyRemoveItems)
	}

	return nil
}

func (v *keywordValidators) ValidateRef(value any, context *validationContext, path schemaPath) error {
	refStr, ok := value.(string)
	if !ok {
		return context.RaiseErrorWithSimplify("$ref must be a string", path, SimplifyRemoveRef)
	}

	// Handle root reference "#"
	if refStr == "#" {
		context.RefPaths[refStr] = struct{}{}
		return nil
	}

	// Must start with #/$defs/
	if !strings.HasPrefix(refStr, "#/$defs/") {
		return context.RaiseErrorWithSimplify("references must start with #/$defs/", path, SimplifyRemoveRef)
	}

	// Need to point to specific definition
	defName := refStr[len("#/$defs/"):]
	if defName == "" {
		return context.RaiseErrorWithSimplify("definition name cannot be empty", path, SimplifyRemoveRef)
	}

	// Check if $defs exists
	if _, hasDefs := context.SchemaRoot[Defs]; !hasDefs {
		return context.RaiseErrorWithSimplify(fmt.Sprintf("$defs not found for reference: %s", refStr), path, SimplifyRemoveRef)
	}

	// Get parent schema
	var parentSchema SchemaDict
	is_root := false
	if path.String() == Ref { // root level
		parentSchema = context.SchemaRoot
		is_root = true
	} else {
		parentPath := path.Parent()
		var err error
		parentSchema, err = v.utils.ResolveSubschema(context.SchemaRoot, parentPath, context, path)
		if err != nil || parentSchema == nil {
			return context.RaiseError(fmt.Sprintf("invalid path: %s", parentPath), path)
		}
	}

	// Check if $ref is allowed at the same level as other keywords
	for key := range parentSchema {
		if key == Ref {
			continue
		}

		if is_root && TopLevelOnlyKeywords[key] {
			continue
		}

		if v.config.IsStrict() || v.config.IsTest() {
			if !CommonKeywords[key] {
				return context.RaiseErrorWithSimplify(
					fmt.Sprintf("keyword '%s' is not allowed at the same level as $ref", key),
					path.StringWithoutLast(),
					SimplifyDefault,
				)
			}
		}
	}

	// Add to ref_paths
	context.RefPaths[refStr] = struct{}{}

	return nil
}

func (v *keywordValidators) ValidateDescription(value any, context *validationContext, path schemaPath) error {
	_, ok := value.(string)
	if !ok {
		return context.RaiseErrorWithSimplify("description must be a string", path, SimplifyRemoveDescription)
	}
	return nil
}

func (v *keywordValidators) ValidateAnyOf(value any, context *validationContext, path schemaPath) error {
	anyOf, ok := value.(SchemaList)
	if !ok {
		return context.RaiseErrorWithSimplify("anyOf must be an array", path, SimplifyRemoveAnyOf)
	}

	if len(anyOf) == 0 || len(anyOf) > v.config.MaxAnyOfItems {
		return context.RaiseErrorWithSimplify(fmt.Sprintf("anyOf must have 1-%d items", v.config.MaxAnyOfItems), path, SimplifyRemoveAnyOf)
	}

	for _, schema := range anyOf {
		_, ok := schema.(SchemaDict)
		if !ok {
			return context.RaiseErrorWithSimplify("schema in anyOf must be an object", path, SimplifyRemoveAnyOf)
		}
	}

	// Get parent schema
	var parentSchema SchemaDict
	is_root := false
	if path.String() == AnyOf { // root level
		parentSchema = context.SchemaRoot
		is_root = true
	} else {
		parentPath := path.Parent()

		var err error
		parentSchema, err = v.utils.ResolveSubschema(context.SchemaRoot, parentPath, context, path)
		if err != nil || parentSchema == nil {
			return context.RaiseError(fmt.Sprintf("invalid path: %s", parentPath), path)
		}
	}

	// Check if anyOf is allowed at the same level as other keywords
	for key := range parentSchema {
		if key == AnyOf {
			continue
		}

		if is_root && TopLevelOnlyKeywords[key] {
			continue
		}

		if v.config.IsStrict() || v.config.IsTest() {
			if !CommonKeywords[key] {
				return context.RaiseErrorWithSimplify(
					fmt.Sprintf("keyword '%s' is not allowed at the same level as anyOf", key),
					path.StringWithoutLast(),
					SimplifyDefault,
				)
			}
		}
	}

	return nil
}

func (v *keywordValidators) ValidateAdditionalProperties(value any, context *validationContext, path schemaPath) error {
	switch value.(type) {
	case bool:
		return nil
	case SchemaDict:
		return nil
	default:
		return context.RaiseErrorWithSimplify("additionalProperties must be a boolean or an object", path, SimplifyRemoveAdditionalProperties)
	}
}

func (v *keywordValidators) ValidateDefs(value any, context *validationContext, path schemaPath) error {
	defs, ok := value.(SchemaDict)
	if !ok {
		return context.RaiseErrorWithSimplify("$defs must be an object", path, SimplifyRemoveDefs)
	}

	// Validate each definition
	for defName, defSchema := range defs {
		if len(defName) == 0 {
			return context.RaiseErrorWithSimplify("$defs property name cannot be empty", path, SimplifyRemoveDefsEmptySubSchema)
		}

		if strings.Contains(defName, "/") {
			return context.RaiseErrorWithSimplify(fmt.Sprintf("$defs property name '%s' cannot contain '/' character", defName), path, SimplifyRemoveDefsEmptySubSchema)
		}

		if defSchema == nil {
			return context.RaiseErrorWithSimplify(fmt.Sprintf("$defs schema must be object: %s", defName), path, SimplifyRemoveDefs)
		}

		defSchemaObj, ok := defSchema.(SchemaDict)
		if !ok {
			return context.RaiseErrorWithSimplify(fmt.Sprintf("$defs schema must be object: %s", defName), path, SimplifyRemoveDefs)
		}

		// iterate defSchemaObj, check if any key contains '/' character
		if err := v.validateNoSlashInKeys(defSchemaObj, context, path.Append(defName)); err != nil {
			return err
		}
	}

	return nil
}

func (v *keywordValidators) validateNoSlashInKeys(value any, context *validationContext, path schemaPath) error {
	schemaDict, ok := value.(SchemaDict)
	if !ok {
		return context.RaiseErrorWithSimplify("schema must be an object", path, SimplifyRemoveSubSchema)
	}

	for key, value := range schemaDict {
		if strings.Contains(key, "/") {
			return context.RaiseErrorWithSimplify(fmt.Sprintf("$defs property name '%s' cannot contain '/' character", key), path.Parent(), SimplifyRemoveDefsEmptySubSchema)
		}

		if subSchema, ok := value.(SchemaDict); ok {
			if err := v.validateNoSlashInKeys(subSchema, context, path.Append(key)); err != nil {
				return err
			}
		}

		if subSchemaList, ok := value.(SchemaList); ok {
			for i, item := range subSchemaList {
				if subSchema, ok := item.(SchemaDict); ok {
					if err := v.validateNoSlashInKeys(subSchema, context, path.Append(fmt.Sprintf("%s[%d]", key, i))); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (v *keywordValidators) ValidateID(value any, context *validationContext, path schemaPath) error {
	if value == nil {
		return nil
	}

	_, ok := value.(string)
	if !ok {
		return context.RaiseErrorWithSimplify("$id must be a string", path, SimplifyRemoveID)
	}

	// if idStr == "" {
	// 	return context.RaiseError("$id cannot be empty", path)
	// }

	return nil
}

func (v *keywordValidators) ValidatePattern(value any, context *validationContext, path schemaPath) error {
	if value == nil {
		return nil
	}

	_, ok := value.(string)
	if !ok {
		return context.RaiseErrorWithSimplify("pattern must be a string", path, SimplifyRemovePattern)
	}

	return nil
}

func (v *keywordValidators) ValidateDefault(value any, context *validationContext, path schemaPath) error {
	// Default keyword validation is currently not implemented
	return nil
}

func (v *keywordValidators) ValidateLengthRange(value any, context *validationContext, path schemaPath) error {
	schema, ok := value.(SchemaDict)
	if !ok {
		return context.RaiseErrorWithSimplify("minLength and maxLength parent schema must be an object", path, SimplifyRemoveConstraints)
	}

	minLength, hasMinLength := schema[MinLength]
	maxLength, hasMaxLength := schema[MaxLength]

	// Check type and non-negative for minLength if present
	if hasMinLength && minLength != nil {
		switch val := minLength.(type) {
		case float64:
			if val < 0 {
				return context.RaiseErrorWithSimplify("minLength must be non-negative", path, SimplifyNegativeVal)
			}
			if err := v.utils.IsValidInteger(val, context, path); err != nil {
				return err
			}
		default:
			return context.RaiseErrorWithSimplify("minLength must be an integer", path, SimplifyDefault)
		}
	}

	// Check type and non-negative for maxLength if present
	if hasMaxLength && maxLength != nil {
		switch val := maxLength.(type) {
		case float64:
			if val < 0 {
				return context.RaiseErrorWithSimplify("maxLength must be non-negative", path, SimplifyNegativeVal)
			}
			if err := v.utils.IsValidInteger(val, context, path); err != nil {
				return err
			}
		default:
			return context.RaiseErrorWithSimplify("maxLength must be an integer", path, SimplifyDefault)
		}
	}

	// Only check range if both are present
	if hasMinLength && hasMaxLength && minLength != nil && maxLength != nil {
		minVal := minLength.(float64)
		maxVal := maxLength.(float64)
		if minVal > maxVal {
			return context.RaiseErrorWithSimplify(
				fmt.Sprintf("minLength (%v) cannot be greater than maxLength (%v)", minLength, maxLength),
				path.Append(MinLength), SimplifyRemoveConstraints,
			)
		}
	}

	return nil
}

func (v *keywordValidators) ValidateNumericRange(value any, context *validationContext, path schemaPath) error {
	schema, ok := value.(SchemaDict)
	if !ok {
		return context.RaiseErrorWithSimplify("minimum and maximum parent schema must be an object", path, SimplifyRemoveConstraints)
	}

	minimum, hasMinimum := schema[Minimum]
	maximum, hasMaximum := schema[Maximum]
	schemaType, hasType := schema[Type]

	if !hasType {
		return nil
	}

	isInteger := false
	switch t := schemaType.(type) {
	case string:
		if t == Integer {
			isInteger = true
		}
	case SchemaList:
		for _, t := range t {
			if t == Integer {
				isInteger = true
				break
			}
		}
	}

	if hasMinimum && minimum != nil {
		if isInteger {
			switch val := minimum.(type) {
			case float64:
				if err := v.utils.IsValidInteger(val, context, path); err != nil {
					return err
				}
			default:
				return context.RaiseErrorWithSimplify("minimum must be an integer", path, SimplifyDefault)
			}
		} else {
			switch val := minimum.(type) {
			case float64:
				if err := v.utils.IsValidNumber(val, context, path); err != nil {
					return err
				}
			default:
				return context.RaiseErrorWithSimplify("minimum must be a number", path, SimplifyDefault)
			}
		}
	}

	if hasMaximum && maximum != nil {
		if isInteger {
			switch val := maximum.(type) {
			case float64:
				if err := v.utils.IsValidInteger(val, context, path); err != nil {
					return err
				}
			default:
				return context.RaiseErrorWithSimplify("maximum must be an integer", path, SimplifyDefault)
			}
		} else {
			switch val := maximum.(type) {
			case float64:
				if err := v.utils.IsValidNumber(val, context, path); err != nil {
					return err
				}
			default:
				return context.RaiseErrorWithSimplify("maximum must be a number", path, SimplifyDefault)
			}
		}
	}

	// Only validate minimum <= maximum if both values are provided
	if hasMinimum && hasMaximum && minimum != nil && maximum != nil {
		minVal := minimum.(float64)
		maxVal := maximum.(float64)
		if minVal > maxVal {
			return context.RaiseErrorWithSimplify(
				fmt.Sprintf("minimum (%v) cannot be greater than maximum (%v)", minimum, maximum),
				path, SimplifyRemoveConstraints,
			)
		}
	}

	return nil
}

func (v *keywordValidators) ValidateItemsRange(value any, context *validationContext, path schemaPath) error {
	schema, ok := value.(SchemaDict)
	if !ok {
		return context.RaiseErrorWithSimplify("minItems and maxItems parent schema must be an object", path, SimplifyRemoveConstraints)
	}

	minItems, hasMinItems := schema[MinItems]
	maxItems, hasMaxItems := schema[MaxItems]

	// Check type and non-negative for minItems if present
	if hasMinItems && minItems != nil {
		switch val := minItems.(type) {
		case float64:
			if val < 0 {
				return context.RaiseErrorWithSimplify("minItems must be non-negative", path, SimplifyNegativeVal)
			}
			if err := v.utils.IsValidInteger(val, context, path); err != nil {
				return err
			}
		default:
			return context.RaiseErrorWithSimplify("minItems must be an integer", path, SimplifyDefault)
		}
	}

	// Check type and non-negative for maxItems if present
	if hasMaxItems && maxItems != nil {
		switch val := maxItems.(type) {
		case float64:
			if val < 0 {
				return context.RaiseErrorWithSimplify("maxItems must be non-negative", path, SimplifyNegativeVal)
			}
			if err := v.utils.IsValidInteger(val, context, path); err != nil {
				return err
			}
		default:
			return context.RaiseErrorWithSimplify("maxItems must be an integer", path, SimplifyDefault)
		}
	}

	// Only check range if both are present
	if hasMinItems && hasMaxItems && minItems != nil && maxItems != nil {
		minVal := minItems.(float64)
		maxVal := maxItems.(float64)
		if minVal > maxVal {
			return context.RaiseErrorWithSimplify(
				fmt.Sprintf("minItems (%v) cannot be greater than maxItems (%v)", minItems, maxItems),
				path.Append(MinItems), SimplifyRemoveConstraints,
			)
		}
	}

	return nil
}
