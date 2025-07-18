package walle

type SchemaList = []any
type SchemaDict = map[string]any

const (
	Type                 = "type"
	Properties           = "properties"
	AdditionalProperties = "additionalProperties"
	Items                = "items"
	Enum                 = "enum"
	Required             = "required"
	AnyOf                = "anyOf"
	Description          = "description"
	Defs                 = "$defs"
	Ref                  = "$ref"
	Title                = "title"
	Id                   = "$id"
	Default              = "default"
	Pattern              = "pattern"
	MaxLength            = "maxLength"
	MinLength            = "minLength"
	Maximum              = "maximum"
	Minimum              = "minimum"
	MaxItems             = "maxItems"
	MinItems             = "minItems"
	String               = "string"
	Number               = "number"
	Integer              = "integer"
	Boolean              = "boolean"
	Null                 = "null"
	Array                = "array"
	Object               = "object"
	Root                 = "root"
)

var (
	// Valid types and supported types
	ValidTypes = map[string]bool{
		"string":  true,
		"number":  true,
		"integer": true,
		"boolean": true,
		"null":    true,
		"array":   true,
		"object":  true,
	}

	// Common keywords
	CommonKeywords = map[string]bool{
		"description": true,
		"title":       true,
	}

	// Allowed keywords for different types
	ObjectAllowedKeywords = mergeMaps(map[string]bool{
		"type":                 true,
		"properties":           true,
		"required":             true,
		"additionalProperties": true,
		"$ref":                 true,
		"anyOf":                true,
		"default":              true,
	}, CommonKeywords)

	ArrayAllowedKeywords = mergeMaps(map[string]bool{
		"type":     true,
		"items":    true,
		"minItems": true,
		"maxItems": true,
		"$ref":     true,
		"default":  true,
	}, CommonKeywords)

	StringAllowedKeywords = mergeMaps(map[string]bool{
		"type":      true,
		"minLength": true,
		"maxLength": true,
		"pattern":   true,
		"enum":      true,
		"default":   true,
	}, CommonKeywords)

	BooleanAllowedKeywords = mergeMaps(map[string]bool{
		"type":    true,
		"enum":    true,
		"default": true,
	}, CommonKeywords)

	NullAllowedKeywords = mergeMaps(map[string]bool{
		"type":    true,
		"enum":    true,
		"default": true,
	}, CommonKeywords)

	NumberAllowedKeywords = mergeMaps(map[string]bool{
		"type":    true,
		"minimum": true,
		"maximum": true,
		"enum":    true,
		"default": true,
	}, CommonKeywords)

	TopLevelOnlyKeywords = map[string]bool{
		"$defs": true,
		"$id":   true,
	}

	// Contexts where $ref is allowed, "properties", "$defs", "root" will be processed independently
	RefAllowedContexts = map[string]bool{
		"additionalProperties": true,
		"anyOf":                true,
		"items":                true,
	}

	// currently supported keywords
	SupportedKeywords = map[string]bool{
		"type":                 true,
		"properties":           true,
		"additionalProperties": true,
		"items":                true,
		"enum":                 true,
		"required":             true,
		"anyOf":                true,
		"description":          true,
		"$defs":                true,
		"$ref":                 true,
		"title":                true,
		"$id":                  true,
		"default":              true,
	}

	// will be supported in the future
	FutureKeywords = map[string]bool{
		"maxLength": true,
		"minLength": true,
		"maximum":   true,
		"minimum":   true,
		"maxItems":  true,
		"minItems":  true,
		"pattern":   true,
	}

	InvalidPropertyNames = map[string]bool{
		"$defs":                true,
		"$ref":                 true,
		"anyOf":                true,
		"required":             true,
		"additionalProperties": true,
	}
)

func mergeMaps(maps ...map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
