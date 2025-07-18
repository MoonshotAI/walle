package walle

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type KeywordValidatorFunc func(any, *validationContext, schemaPath) error

type schemaValidator struct {
	context              *validationContext
	keywordValidators    map[string]KeywordValidatorFunc
	validateLengthRange  KeywordValidatorFunc
	validateNumericRange KeywordValidatorFunc
	validateItemsRange   KeywordValidatorFunc
	defDepths            map[string]int
	totalPropKeys        int
	utils                *validateUtils
	config               SchemaValidatorConfig
}

func newSchemaValidator(options ...SchemaValidatorOption) *schemaValidator {
	config := DefaultValidatorConfig()
	for _, option := range options {
		option(&config)
	}

	kv := newKeywordValidators(&config)
	validator := &schemaValidator{
		context:           newValidationContext(),
		keywordValidators: make(map[string]KeywordValidatorFunc),
		defDepths:         make(map[string]int),
		totalPropKeys:     0,
		utils:             &validateUtils{},
		config:            config,
	}

	// Register keyword validators
	validator.keywordValidators[Type] = kv.ValidateType
	validator.keywordValidators[Properties] = kv.ValidateProperties
	validator.keywordValidators[Required] = kv.ValidateRequired
	validator.keywordValidators[Enum] = kv.ValidateEnum
	validator.keywordValidators[Items] = kv.ValidateItems
	validator.keywordValidators[Ref] = kv.ValidateRef
	validator.keywordValidators[Description] = kv.ValidateDescription
	validator.keywordValidators[AnyOf] = kv.ValidateAnyOf
	validator.keywordValidators[AdditionalProperties] = kv.ValidateAdditionalProperties
	validator.keywordValidators[Defs] = kv.ValidateDefs
	validator.keywordValidators[Id] = kv.ValidateID
	validator.keywordValidators[Pattern] = kv.ValidatePattern
	validator.keywordValidators[Default] = kv.ValidateDefault
	validator.validateLengthRange = kv.ValidateLengthRange
	validator.validateNumericRange = kv.ValidateNumericRange
	validator.validateItemsRange = kv.ValidateItemsRange

	return validator
}

// Reset resets the validator state
func (v *schemaValidator) Reset() {
	v.context = newValidationContext()
	v.defDepths = make(map[string]int)
	v.totalPropKeys = 0
	// won't reset config
}

func (v *schemaValidator) MakePath(base schemaPath, parts ...string) schemaPath {
	if base.IsRoot() {
		return newSchemaPathFromParts(parts)
	}
	return base.Append(parts...)
}

func (v *schemaValidator) CheckAnyOfConflicts(schema SchemaDict, path schemaPath) error {
	anyOf, ok := schema[AnyOf]
	if !ok {
		return nil
	}

	// Get outer keywords (excluding common keywords and anyOf itself)
	outerKeywords := make(map[string]struct{})
	for k := range schema {
		if k != AnyOf && k != Description && k != Title {
			outerKeywords[k] = struct{}{}
		}
	}

	// Check each anyOf branch
	anyOfSchemas, ok := anyOf.(SchemaList)
	if !ok {
		return v.context.RaiseErrorWithSimplify("anyOf must be an array", path.Append(AnyOf), SimplifyRemoveAnyOf)
	}

	for _, subschema := range anyOfSchemas {
		schemaObj, ok := subschema.(SchemaDict)
		if !ok {
			return v.context.RaiseErrorWithSimplify("schema in anyOf must be an object", path.Append(AnyOf), SimplifyRemoveAnyOf)
		}

		// Check branch keywords against outer keywords
		branchKeywords := make(map[string]struct{})
		for k := range schemaObj {
			if k != Description && k != Title {
				branchKeywords[k] = struct{}{}
			}
		}

		var conflicts []string
		for k := range branchKeywords {
			if _, exists := outerKeywords[k]; exists {
				conflicts = append(conflicts, k)
			}
		}

		if len(conflicts) > 0 {
			return v.context.RaiseErrorWithSimplify(
				fmt.Sprintf("conflicting keywords found in anyOf with parent: %s",
					strings.Join(conflicts, ", ")),
				path, SimplifyRemoveParentSchema,
			)
		}
	}

	return nil
}

// TraverseSchema traverses and validates a schema
func (v *schemaValidator) TraverseSchema(schema SchemaDict, path schemaPath, currentDepth int) (int, error) {
	maxDepth := currentDepth

	if currentDepth > v.config.MaxSchemaDepth {
		return currentDepth, v.context.RaiseError(fmt.Sprintf("schema depth exceeds maximum limit of %d", v.config.MaxSchemaDepth), path)
	}

	if schema == nil {
		return currentDepth, nil
	}

	// Verify if it contains unsupported keywords
	var unsupported []string
	for k := range schema {
		if _, ok := SupportedKeywords[k]; !ok && !FutureKeywords[k] {
			unsupported = append(unsupported, k)
		}
	}

	if len(unsupported) > 0 && (v.config.IsStrict() || v.config.IsTest()) {
		return currentDepth, v.context.RaiseErrorWithSimplify(fmt.Sprintf("unsupported keywords: %s", strings.Join(unsupported, ", ")), path, SimplifyDefault)
	}

	// Process $defs first
	if defs, ok := schema[Defs].(SchemaDict); ok {
		keywords := make([]string, 0, len(defs))
		for keyword := range defs {
			keywords = append(keywords, keyword)
		}
		sort.Strings(keywords)
		for _, keyword := range keywords {
			value := defs[keyword]
			defSchemaObj, ok := value.(SchemaDict)
			if !ok {
				return currentDepth, v.context.RaiseErrorWithSimplify("$defs schema must be object", path.Append(Defs), SimplifyRemoveDefs)
			}
			depth, err := v.TraverseSchema(
				defSchemaObj,
				v.MakePath(path, Defs, keyword),
				currentDepth,
			)
			if err != nil {
				return currentDepth, err
			}
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	// Check type and anyOf/ref conflicts
	if _, hasType := schema[Type]; hasType {
		if _, hasAnyOf := schema[AnyOf]; hasAnyOf {
			return currentDepth, v.context.RaiseErrorWithSimplify(
				"when using anyOf, type should be defined in anyOf items instead of the parent schema",
				path, SimplifyRemoveType,
			)
		}
		if _, hasRef := schema[Ref]; hasRef {
			return currentDepth, v.context.RaiseErrorWithSimplify(
				"when using $ref, type should be defined in the referenced schema instead of the parent schema",
				path, SimplifyRemoveType,
			)
		}

		if typeList, ok := schema[Type].(SchemaList); ok && len(typeList) > 1 {
			if _, hasEnum := schema[Enum]; hasEnum {
				for _, t := range typeList {
					if typeStr, ok := t.(string); ok {
						if typeStr == Object || typeStr == Array {
							return currentDepth, v.context.RaiseErrorWithSimplify(
								fmt.Sprintf("type %s is not allowed in combination with enum", typeStr),
								path, SimplifyRemoveParentSchema,
							)
						}
					}
				}
			}
		}
	}
	// else {
	// Empty schema {} is valid
	// isAnySchema := false
	// isAdditional := false
	// if path.Last() == AdditionalProperties {
	// 	isAdditional = true
	// }

	// if len(schema) == 0 && isAdditional {
	// 	isAnySchema = true
	// }

	// if _, hasAnyOf := schema[AnyOf]; !hasAnyOf {
	// 	if _, hasRef := schema[Ref]; !hasRef {
	// 		if !isAnySchema {
	// 			return currentDepth, v.context.RaiseErrorWithSimplify("type need to be defined explicitly", path, SimplifyRemoveSubSchema)
	// 		}
	// 	}
	// }
	// }

	// Type keyword validation
	if err := v.validateTypeAndKeywords(schema, path); err != nil {
		return currentDepth, err
	}

	// Other keywords validation
	keywords := make([]string, 0, len(schema))
	for keyword := range schema {
		keywords = append(keywords, keyword)
	}
	sort.Strings(keywords)
	for _, keyword := range keywords {
		value := schema[keyword]
		if validator, ok := v.keywordValidators[keyword]; ok {
			if err := validator(value, v.context, v.MakePath(path, keyword)); err != nil {
				return currentDepth, err
			}
		}
	}

	// Check anyOf conflicts
	if err := v.CheckAnyOfConflicts(schema, path); err != nil {
		return currentDepth, err
	}

	// process $ref depth
	if ref, ok := schema[Ref].(string); ok {
		refDepth, exists := v.defDepths[ref]
		if !exists {
			refDepth = 0
		}
		if currentDepth+refDepth > maxDepth {
			maxDepth = currentDepth + refDepth
		}
	}

	// Recursive verify sub-schema
	// properties
	if props, ok := schema[Properties].(SchemaDict); ok {
		propsDepth := currentDepth + 1
		v.totalPropKeys += len(props)
		if v.totalPropKeys > v.config.MaxTotalPropertiesKeysNum {
			return currentDepth, v.context.RaiseError(
				fmt.Sprintf("total number of properties keys(%d) across all objects exceeds maximum limit of %d",
					v.totalPropKeys, v.config.MaxTotalPropertiesKeysNum),
				path,
			)
		}

		for propName, propSchema := range props {
			propSchemaObj, ok := propSchema.(SchemaDict)
			if !ok {
				continue
			}
			depth, err := v.TraverseSchema(propSchemaObj, v.MakePath(path, Properties, propName), propsDepth)
			if err != nil {
				return currentDepth, err
			}
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	// items
	if items, ok := schema[Items].(SchemaDict); ok {
		depth, err := v.TraverseSchema(items, v.MakePath(path, Items), currentDepth)
		if err != nil {
			return currentDepth, err
		}
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	// additionalProperties
	if addProps, ok := schema[AdditionalProperties].(SchemaDict); ok {
		depth, err := v.TraverseSchema(
			addProps,
			v.MakePath(path, AdditionalProperties),
			currentDepth,
		)
		if err != nil {
			return currentDepth, err
		}
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	// anyOf
	if anyOf, ok := schema[AnyOf].(SchemaList); ok {
		for i, subSchema := range anyOf {
			subSchemaObj, ok := subSchema.(SchemaDict)
			if !ok {
				continue
			}
			depth, err := v.TraverseSchema(
				subSchemaObj,
				v.MakePath(path, fmt.Sprintf("anyOf{%d}", i)),
				currentDepth,
			)
			if err != nil {
				return currentDepth, err
			}
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	return maxDepth, nil
}

func (v *schemaValidator) validateTypeAndKeywords(schema SchemaDict, path schemaPath) error {
	if typeVal, ok := schema[Type]; ok {
		var types []string

		switch t := typeVal.(type) {
		case string:
			if !ValidTypes[t] {
				return v.context.RaiseErrorWithSimplify("invalid type", path, SimplifyRemoveParentSchema)
			}
			types = append(types, t)
		case SchemaList:
			for _, item := range t {
				if str, ok := item.(string); !ok || !ValidTypes[str] {
					return v.context.RaiseErrorWithSimplify("invalid type in type array", path, SimplifyRemoveParentSchema)
				}
				types = append(types, item.(string))
			}
			if len(types) == 0 {
				return v.context.RaiseErrorWithSimplify("type array cannot be empty", path, SimplifyRemoveParentSchema)
			}
		}

		var chooseType string
		if len(types) > 1 {
			allowedEnum := false
			if len(types) == 2 && schema[Enum] != nil {
				for _, t := range types {
					if t == Object || t == Array {
						return v.context.RaiseErrorWithSimplify("object and array cannot be used in combination with multiple types", path, SimplifyRemoveParentSchema)
					}
				}
				allowedEnum = true
			}

			for k := range schema {
				if path.IsRoot() {
					if !TopLevelOnlyKeywords[k] && !CommonKeywords[k] && k != Type && !allowedEnum {
						return v.context.RaiseErrorWithSimplify(fmt.Sprintf("keyword %s is not allowed in combination with multiple types", k), path, SimplifyRemoveParentSchema)
					}
				} else {
					if !CommonKeywords[k] && k != Type && !allowedEnum {
						return v.context.RaiseErrorWithSimplify(fmt.Sprintf("keyword %s is not allowed in combination with multiple types", k), path, SimplifyRemoveParentSchema)
					}
				}
			}
		}

		if len(types) >= 1 && (v.config.IsStrict() || v.config.IsTest()) {
			// FIXME: Check every type later.
			chooseType = types[0]
			// Check $defs and $id are only at top level
			if !path.IsRoot() {
				for k := range schema {
					if TopLevelOnlyKeywords[k] {
						return v.context.RaiseErrorWithSimplify(fmt.Sprintf("keyword %s must be at root level", k), path, SimplifyDefault)
					}
				}
			}

			allowedKeywords := make(map[string]struct{})
			if path.IsRoot() {
				for k := range TopLevelOnlyKeywords {
					allowedKeywords[k] = struct{}{}
				}
			}

			switch chooseType {
			case Object:
				for k := range ObjectAllowedKeywords {
					allowedKeywords[k] = struct{}{}
				}
			case Array:
				if err := v.validateItemsRange(schema, v.context, path); err != nil {
					return err
				}
				for k := range ArrayAllowedKeywords {
					allowedKeywords[k] = struct{}{}
				}
			case String:
				if err := v.validateLengthRange(schema, v.context, path); err != nil {
					return err
				}
				for k := range StringAllowedKeywords {
					allowedKeywords[k] = struct{}{}
				}
			case Number, Integer:
				if err := v.validateNumericRange(schema, v.context, path); err != nil {
					return err
				}
				for k := range NumberAllowedKeywords {
					allowedKeywords[k] = struct{}{}
				}
			case Boolean:
				for k := range BooleanAllowedKeywords {
					allowedKeywords[k] = struct{}{}
				}
			case Null:
				for k := range NullAllowedKeywords {
					allowedKeywords[k] = struct{}{}
				}
			default:
				return v.context.RaiseErrorWithSimplify(fmt.Sprintf("invalid type: %s", chooseType), path, SimplifyRemoveParentSchema)
			}

			var invalidKeys []string
			for k := range schema {
				if _, exists := allowedKeywords[k]; !exists {
					invalidKeys = append(invalidKeys, k)
				}
			}

			if len(invalidKeys) > 0 {
				return v.context.RaiseErrorWithSimplify(
					fmt.Sprintf("invalid keywords: %s", strings.Join(invalidKeys, ", ")),
					path, SimplifyDefault,
				)
			}
		}
	}

	return nil
}

// Validate validates a JSON schema
func (v *schemaValidator) Validate(schema any) error {
	switch s := schema.(type) {
	case string:
		var schemaDict SchemaDict
		err := json.Unmarshal([]byte(s), &schemaDict)
		if err != nil {
			switch e := err.(type) {
			case *json.SyntaxError:
				return NewUnmarshalError(fmt.Errorf("JSON syntax error at offset %d: %s", e.Offset, e.Error()))
			case *json.UnmarshalTypeError:
				return NewUnmarshalError(fmt.Errorf("JSON type error at offset %d: expected %s but got %s",
					e.Offset, e.Type, e.Value))
			default:
				return NewUnmarshalError(err)
			}
		}
		return v.validateSchemaDict(schemaDict)
	case SchemaDict:
		return v.validateSchemaDict(s)
	case Schema:
		return v.validateSchemaDict(s)
	default:
		return v.context.RaiseError("input schema must be a string or map", rootSchemaPath)
	}
}

func (v *schemaValidator) CanonicalWithMaxAttempts(schema Schema, maxAttempts int) (string, error) {
	currentSchema := schema

	var rawErr error
	for i := 0; i < maxAttempts; i++ {
		err := v.Validate(currentSchema)
		if i == 0 {
			rawErr = err
		}
		if err == nil {
			schemaStr, _ := json.Marshal(currentSchema)
			return string(schemaStr), rawErr
		}

		if schemaErr, ok := err.(*SchemaError); ok && schemaErr.SimplifyFunc != nil {
			pathObj := newSchemaPath(schemaErr.Path)
			currentSchema = schemaErr.SimplifyFunc(currentSchema, pathObj)
		} else {
			return "{}", rawErr
		}
	}

	return "{}", rawErr
}

func (v *schemaValidator) validateSchemaDict(schema SchemaDict) error {
	// Reset state
	v.Reset()

	if schema == nil {
		return v.context.RaiseError("schema must be a dict", rootSchemaPath)
	}

	// Empty schema {} is valid
	if len(schema) == 0 {
		return nil
	}

	v.context.SchemaRoot = schema

	// Verify schema string length
	if v.utils.CalculateSchemaSize(schema) > v.config.MaxSchemaSize {
		return v.context.RaiseError("schema exceeds maximum allowed size", rootSchemaPath)
	}

	// Precompute defs depth
	v.defDepths = v.CalculateDefDepths()

	// Verify schema
	maxDepth, err := v.TraverseSchema(schema, rootSchemaPath, 0)
	if err != nil {
		return err
	}

	if maxDepth > v.config.MaxSchemaDepth {
		return v.context.RaiseError(fmt.Sprintf("schema depth exceeds maximum limit of %d", v.config.MaxSchemaDepth), rootSchemaPath)
	}

	// Verify ref path is valid
	return v.PostValidateRefs()
}

// PostValidateRefs validates all references after schema traversal
func (v *schemaValidator) PostValidateRefs() error {
	// Verify all ref paths exist
	for refPath := range v.context.RefPaths {
		if _, err := v.utils.ResolveRef(v.context.SchemaRoot, refPath, v.context, rootSchemaPath); err != nil {
			return v.context.RaiseError(fmt.Sprintf("invalid $ref path: %s", refPath), rootSchemaPath)
		}
	}

	// first check root schema whether it can terminate
	needCheckTermination := true
	if terminates, err := v.CheckRefTermination(v.context.SchemaRoot, make(map[string]struct{}), rootSchemaPath); err == nil {
		if terminates {
			needCheckTermination = false
		}
	}
	// Traverse and check all references
	return v.TraverseAndCheckRefs(v.context.SchemaRoot, needCheckTermination, nil, rootSchemaPath)
}

// TraverseAndCheckRefs traverses the schema and checks all references
func (v *schemaValidator) TraverseAndCheckRefs(schema SchemaDict, needCheckTermination bool, requiredList SchemaList, path schemaPath) error {
	if schema == nil {
		return nil
	}

	if _, hasRef := schema[Ref]; hasRef {
		expanded, err := v.ExpandRef(schema, make(map[string]struct{}), path)
		if err != nil {
			return err
		}

		if err := v.CheckRefContext(schema, expanded, path); err != nil {
			return err
		}

		// Check that all refs can be terminated
		if needCheckTermination {
			terminates, err := v.CheckRefTermination(expanded, make(map[string]struct{}), path)
			if err != nil {
				return err
			}
			if !terminates {
				return v.context.RaiseError("detected infinite recursion without termination condition", path)
			}
		}
	}

	for key, value := range schema {
		if key == Defs {
			continue
		}

		var newPath schemaPath
		if path.IsRoot() {
			newPath = schemaPath{Parts: []string{key}}
		} else {
			newPath = path.Append(key)
		}

		switch schemaValue := value.(type) {
		case SchemaDict:
			// skip some conditions that do not need to check termination
			if key == Properties {
				if required, ok := schema[Required]; ok {
					if requiredList, ok := required.(SchemaList); ok {
						if len(requiredList) == 0 {
							needCheckTermination = false
						}
					}
				}
			} else if key == AdditionalProperties {
				needCheckTermination = false
			} else if len(requiredList) > 0 {
				findRequired := false
				for _, req := range requiredList {
					if key == req {
						findRequired = true
						break
					}
				}
				if !findRequired {
					needCheckTermination = false
				}
			}

			if err := v.TraverseAndCheckRefs(schemaValue, needCheckTermination, requiredList, newPath); err != nil {
				return err
			}
		case SchemaList:
			if key == AnyOf {
				for i, item := range schemaValue {
					if itemSchema, ok := item.(SchemaDict); ok {
						if err := v.TraverseAndCheckRefs(
							itemSchema,
							needCheckTermination,
							nil,
							newPath.ModifyAnyOfPart(i),
						); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

func (v *schemaValidator) CalculateDefDepths() map[string]int {
	defDepths := make(map[string]int)

	var calculateDepthsRecursive func(schema SchemaDict, currentPath string, visitedRefs map[string]struct{}) int
	calculateDepthsRecursive = func(schema SchemaDict, currentPath string, visitedRefs map[string]struct{}) int {
		// The depth of basic type or empty schema is 0
		if len(schema) == 0 {
			return 0
		}

		// Record the depth of current path
		defDepths[currentPath] = 0 // Initial depth is 0
		maxDepth := 0

		if ref, ok := schema[Ref].(string); ok {
			if _, exists := visitedRefs[ref]; !exists {
				visitedRefs[ref] = struct{}{}
				resolved, err := v.utils.ResolveRef(v.context.SchemaRoot, ref, v.context, rootSchemaPath)
				if err == nil && resolved != nil {
					maxDepth = calculateDepthsRecursive(resolved, ref, visitedRefs)
				}
			}
		}

		if props, ok := schema[Properties].(SchemaDict); ok {
			propsDepth := 1
			for propName, propSchema := range props {
				propSchemaObj, ok := propSchema.(SchemaDict)
				if !ok {
					continue
				}
				propPath := fmt.Sprintf("%s/properties/%s", currentPath, propName)

				// Create a new copy of visited refs for each property
				propVisited := make(map[string]struct{})
				for k, v := range visitedRefs {
					propVisited[k] = v
				}

				subDepth := calculateDepthsRecursive(propSchemaObj, propPath, propVisited)
				if 1+subDepth > propsDepth {
					propsDepth = 1 + subDepth
				}
			}
			if propsDepth > maxDepth {
				maxDepth = propsDepth
			}
		}

		if anyOf, ok := schema[AnyOf].(SchemaList); ok {
			for i, subschema := range anyOf {
				subSchemaObj, ok := subschema.(SchemaDict)
				if !ok {
					continue
				}
				subPath := fmt.Sprintf("%s/anyOf/%d", currentPath, i)

				// Create a new copy of visited refs for each anyOf branch
				branchVisited := make(map[string]struct{})
				for k, v := range visitedRefs {
					branchVisited[k] = v
				}

				subDepth := calculateDepthsRecursive(subSchemaObj, subPath, branchVisited)
				if subDepth > maxDepth {
					maxDepth = subDepth
				}
			}
		}

		if addProps, ok := schema[AdditionalProperties].(SchemaDict); ok {
			addPropsPath := fmt.Sprintf("%s/additionalProperties", currentPath)

			// Create a new copy of visited refs
			addPropsVisited := make(map[string]struct{})
			for k, v := range visitedRefs {
				addPropsVisited[k] = v
			}

			subDepth := calculateDepthsRecursive(addProps, addPropsPath, addPropsVisited)
			if subDepth > maxDepth {
				maxDepth = subDepth
			}
		}

		// Update the final depth of current path
		defDepths[currentPath] = maxDepth
		return maxDepth
	}

	// Traverse from $defs
	if defs, ok := v.context.SchemaRoot[Defs].(SchemaDict); ok {
		for defName, defSchema := range defs {
			defSchemaObj, ok := defSchema.(SchemaDict)
			if !ok {
				continue
			}
			basePath := fmt.Sprintf("#/$defs/%s", defName)
			calculateDepthsRecursive(defSchemaObj, basePath, make(map[string]struct{}))
		}
	}

	return defDepths
}

// CheckRefTermination checks if a reference can be terminated
func (v *schemaValidator) CheckRefTermination(schema SchemaDict, visitedRefs map[string]struct{}, path schemaPath) (bool, error) {
	// Non-object/array basic types can terminate
	var checkType string
	if typeVal, ok := schema[Type]; ok {
		switch t := typeVal.(type) {
		case string:
			if t != Object && t != Array {
				return true, nil
			}
			checkType = t
		case SchemaList:
			for _, typ := range t {
				if typeStr, ok := typ.(string); ok && typeStr != Object && typeStr != Array {
					return true, nil
				}

				// should be object or array and len(t) == 1/2
				checkType = typ.(string)
			}
		}
	}

	// Array items if empty can terminate
	if checkType == Array {
		items, exists := schema[Items]
		if !exists || items == nil {
			return true, nil
		}

		if itemsDict, ok := items.(SchemaDict); ok && len(itemsDict) == 0 {
			return true, nil
		}

		// check array items whether it can terminate
		if itemsDict, ok := items.(SchemaDict); ok {
			terminates, err := v.CheckRefTermination(itemsDict, visitedRefs, path.Append(Items))
			if err != nil {
				return false, err
			}
			if terminates {
				return true, nil
			}
		}
	}

	// Object required
	if checkType == Object {
		required, exists := schema[Required]
		// required if empty can terminate
		if !exists || required == nil {
			return true, nil
		}

		if requiredList, ok := required.(SchemaList); ok && len(requiredList) == 0 {
			return true, nil
		}

		// if properties is empty, it can terminate
		props, hasProps := schema[Properties].(SchemaDict)
		if !hasProps || len(props) == 0 {
			return true, nil
		}

		// iterate properties
		requiredSet := make(map[string]struct{})
		for _, req := range required.(SchemaList) {
			if reqStr, ok := req.(string); ok {
				requiredSet[reqStr] = struct{}{}
			}
		}

		for propName, propSchema := range props {
			propSchemaObj, ok := propSchema.(SchemaDict)
			if !ok {
				return false, v.context.RaiseErrorWithSimplify("property schema must be an object", path.Append(propName), SimplifyRemoveProperties)
			}

			// only check required properties
			if _, exists := requiredSet[propName]; !exists {
				continue
			}

			if terminates, err := v.CheckRefTermination(propSchemaObj, visitedRefs, path.Append(Properties, propName)); err != nil {
				return false, err
			} else if terminates {
				return true, nil
			}
		}
	}

	// Check anyOf branches
	if anyOf, ok := schema[AnyOf].(SchemaList); ok {
		// if anyOf is empty, it can terminate
		if len(anyOf) == 0 {
			return true, nil
		}

		allRefs := make(map[string]struct{})
		for i, item := range anyOf {
			itemSchema, ok := item.(SchemaDict)
			if !ok {
				return false, v.context.RaiseErrorWithSimplify("schema in anyOf must be an object", path.Append(AnyOf), SimplifyRemoveAnyOf)
			}

			// Create a new copy of visited refs for each branch
			branchRefs := make(map[string]struct{})
			for k := range visitedRefs {
				branchRefs[k] = struct{}{}
			}

			if terminates, err := v.CheckRefTermination(itemSchema, branchRefs, path.Append(AnyOf, strconv.Itoa(i))); err != nil {
				return false, err
			} else if terminates {
				// Update visited refs with branch refs
				for k := range branchRefs {
					visitedRefs[k] = struct{}{}
				}
				return true, nil
			}

			// Collect all refs from this branch
			for k := range branchRefs {
				allRefs[k] = struct{}{}
			}
		}

		// Update visited refs with all collected refs
		for k := range allRefs {
			visitedRefs[k] = struct{}{}
		}
	}

	// Check ref
	if ref, ok := schema[Ref].(string); ok {
		if _, exists := visitedRefs[ref]; exists {
			return false, nil
		}

		visitedRefs[ref] = struct{}{}
		target, err := v.utils.ResolveRef(v.context.SchemaRoot, ref, v.context, path)
		if err != nil {
			return false, v.context.RaiseError(fmt.Sprintf("invalid $ref path: %s", ref), path)
		}

		return v.CheckRefTermination(target, visitedRefs, path.Append(Ref))
	}

	if len(schema) == 0 {
		return true, nil
	}

	return false, nil
}

// ExpandRef expands a reference
func (v *schemaValidator) ExpandRef(schema SchemaDict, visitedRefs map[string]struct{}, path schemaPath) (SchemaDict, error) {
	if schema == nil || schema[Ref] == nil {
		return schema, nil
	}

	refValue, ok := schema[Ref].(string)
	if !ok {
		return nil, v.context.RaiseErrorWithSimplify("$ref must be a string", path.Append(Ref), SimplifyRemoveRef)
	}

	ref := refValue
	if _, exists := visitedRefs[ref]; exists {
		return schema, nil // Avoid circular references
	}

	visitedRefs[ref] = struct{}{}
	resolved, err := v.utils.ResolveRef(v.context.SchemaRoot, ref, v.context, rootSchemaPath)
	if err != nil {
		return nil, err
	}

	// First expansion
	currentResolved := make(SchemaDict)
	for k, v := range resolved {
		currentResolved[k] = v
	}

	// Keep expanding if only contains $ref and common keywords
	keepKeys := make(SchemaDict)
	for {
		if currentResolved[Ref] == nil {
			break
		}

		keys := make(map[string]struct{})
		for k := range currentResolved {
			keys[k] = struct{}{}
		}

		// Check if there are other keywords besides common keys and $ref
		hasOtherKeys := false
		for k := range keys {
			if k != Ref && !CommonKeywords[k] {
				hasOtherKeys = true
				break
			}
		}

		if hasOtherKeys {
			break
		}

		nextRef, ok := currentResolved[Ref].(string)
		if !ok {
			return nil, v.context.RaiseErrorWithSimplify("$ref must be a string", path.Append(Ref), SimplifyRemoveRef)
		}

		if _, exists := visitedRefs[nextRef]; exists {
			break
		}

		visitedRefs[nextRef] = struct{}{}
		nextResolved, err := v.utils.ResolveRef(v.context.SchemaRoot, nextRef, v.context, rootSchemaPath)
		if err != nil {
			return nil, err
		}

		// Keep common keywords from current level
		for k := range currentResolved {
			if CommonKeywords[k] {
				if _, exists := keepKeys[k]; exists {
					return nil, v.context.RaiseErrorWithSimplify(
						fmt.Sprintf("conflicting keywords found after $ref expansion: %s", k),
						path, SimplifyRemoveParentSchema,
					)
				}
				keepKeys[k] = currentResolved[k]
			}
		}

		currentResolved = nextResolved
	}

	result := currentResolved
	for k, v := range keepKeys {
		result[k] = v
	}

	return result, nil
}

// CheckRefContext checks if expanded ref schema conflicts with parent schema
func (v *schemaValidator) CheckRefContext(parent SchemaDict, refSchema SchemaDict, path schemaPath) error {
	if refSchema == nil {
		return nil
	}

	// First check for direct keyword conflicts
	parentKeywords := make(map[string]struct{})
	for k := range parent {
		if k != Ref {
			parentKeywords[k] = struct{}{}
		}
	}

	var conflicts []string
	for k := range refSchema {
		if _, exists := parentKeywords[k]; exists {
			conflicts = append(conflicts, k)
		}
	}

	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return v.context.RaiseErrorWithSimplify(fmt.Sprintf("conflicting keywords found after $ref expansion: %s", strings.Join(conflicts, ", ")), path, SimplifyRemoveParentSchema)
	}

	// Validate if the parent schema is valid after ref expansion
	_, hasTypeInParent := parent[Type]
	_, hasAnyOfInRef := refSchema[AnyOf]
	_, hasTypeInRef := refSchema[Type]
	_, hasAnyOfInParent := parent[AnyOf]

	if (hasTypeInParent && hasAnyOfInRef) || (hasTypeInRef && hasAnyOfInParent) {
		return v.context.RaiseErrorWithSimplify("invalid schema after $ref expansion: (when using anyOf, type should be defined in anyOf items instead of the parent schema)", path, SimplifyRemoveParentSchema)
	}

	// If refSchema contains anyOf, check for conflicts after expansion
	if anyOfValue, ok := refSchema[AnyOf]; ok {
		if anyOfList, ok := anyOfValue.(SchemaList); ok {
			for _, branchInterface := range anyOfList {
				branch, ok := branchInterface.(SchemaDict)
				if !ok {
					continue
				}

				// Check if keywords in each anyOf branch conflict with parent
				branchKeywords := make(map[string]struct{})
				for k := range branch {
					if k != Description && k != Title {
						branchKeywords[k] = struct{}{}
					}
				}

				var branchConflicts []string
				for k := range parentKeywords {
					if _, exists := branchKeywords[k]; exists && k != AnyOf && k != Description && k != Title {
						branchConflicts = append(branchConflicts, k)
					}
				}

				if len(branchConflicts) > 0 {
					sort.Strings(branchConflicts)
					return v.context.RaiseErrorWithSimplify(fmt.Sprintf("conflicting keywords found in anyOf after ref expansion: %s", strings.Join(branchConflicts, ", ")), path, SimplifyRemoveParentSchema)
				}
			}
		}
	}

	// Check if ref appears in a valid context
	parts := path.Parts

	if len(parts) >= 1 && v.isInRefAllowedContexts(path.Last()) {
		return nil
	} else if len(parts) >= 2 && parts[len(parts)-2] == Properties {
		return nil
	} else if len(parts) == 1 {
		return nil
	} else {
		return v.context.RaiseErrorWithSimplify("$ref is not allowed in this schema.", path, SimplifyRemoveRef)
	}
}

// Helper function to check if a path part is in the allowed contexts
func (v *schemaValidator) isInRefAllowedContexts(part string) bool {
	allowedContexts := map[string]bool{
		AdditionalProperties: true,
		AnyOf:                true,
		Items:                true,
	}

	if allowedContexts[part] {
		return true
	}

	return strings.HasPrefix(part, AnyOf)
}

func parseSchema(jsonStr string) (Schema, error) {
	var schemaDict SchemaDict
	err := json.Unmarshal([]byte(jsonStr), &schemaDict)
	if err != nil {
		switch e := err.(type) {
		case *json.SyntaxError:
			return nil, NewUnmarshalError(fmt.Errorf("JSON syntax error at offset %d: %s", e.Offset, e.Error()))
		case *json.UnmarshalTypeError:
			return nil, NewUnmarshalError(fmt.Errorf("JSON type error at offset %d: expected %s but got %s",
				e.Offset, e.Type, e.Value))
		default:
			return nil, NewUnmarshalError(err)
		}
	}
	return schemaDict, nil
}
