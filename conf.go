package walle

type ValidateLevel string

const (
	ValidateLevelDefault ValidateLevel = "default" // default == ultra
	ValidateLevelLoose   ValidateLevel = "loose"
	ValidateLevelLite    ValidateLevel = "lite"
	ValidateLevelStrict  ValidateLevel = "strict"
	ValidateLevelUltra   ValidateLevel = "ultra"
	ValidateLevelTest    ValidateLevel = "test" // Do not use in production!
)

type SchemaValidatorConfig struct {
	ValidateLevel               ValidateLevel `json:"validateLevel,omitempty"`
	MaxEnumItems                int           `json:"maxEnumItems,omitempty"`
	MaxEnumStringLength         int           `json:"maxEnumStringLength,omitempty"`
	MaxEnumStringCheckThreshold int           `json:"maxEnumStringCheckThreshold,omitempty"`
	MaxAnyOfItems               int           `json:"maxAnyOfItems,omitempty"`
	MaxSchemaDepth              int           `json:"maxSchemaDepth,omitempty"`
	MaxSchemaSize               int           `json:"maxSchemaSize,omitempty"`
	MaxTotalPropertiesKeysNum   int           `json:"maxTotalPropertiesKeysNum,omitempty"`
}

func DefaultValidatorConfig() SchemaValidatorConfig {
	return SchemaValidatorConfig{
		ValidateLevel:               ValidateLevelDefault,
		MaxEnumItems:                500,
		MaxEnumStringLength:         7500,
		MaxEnumStringCheckThreshold: 250,
		MaxAnyOfItems:               100,
		MaxSchemaDepth:              10,
		MaxSchemaSize:               15000,
		MaxTotalPropertiesKeysNum:   1000,
	}
}

type SchemaValidatorOption func(*SchemaValidatorConfig)

func WithValidateLevel(level ValidateLevel) SchemaValidatorOption {
	return func(c *SchemaValidatorConfig) {
		if level == ValidateLevelDefault || level == ValidateLevelLoose || level == ValidateLevelLite || level == ValidateLevelStrict || level == ValidateLevelUltra || level == ValidateLevelTest {
			c.ValidateLevel = level
		}
	}
}

// WithMaxEnumItems set max enum items
func WithMaxEnumItems(max int) SchemaValidatorOption {
	return func(c *SchemaValidatorConfig) {
		if max >= 1 {
			c.MaxEnumItems = max
		}
	}
}

// WithMaxEnumStringLength set max enum string length
func WithMaxEnumStringLength(max int) SchemaValidatorOption {
	return func(c *SchemaValidatorConfig) {
		if max >= 1 {
			c.MaxEnumStringLength = max
		}
	}
}

// WithMaxEnumStringCheckThreshold set max enum string check threshold
func WithMaxEnumStringCheckThreshold(max int) SchemaValidatorOption {
	return func(c *SchemaValidatorConfig) {
		if max >= 1 {
			c.MaxEnumStringCheckThreshold = max
		}
	}
}

// WithMaxAnyOfItems set max anyOf items
func WithMaxAnyOfItems(max int) SchemaValidatorOption {
	return func(c *SchemaValidatorConfig) {
		if max >= 1 {
			c.MaxAnyOfItems = max
		}
	}
}

// WithMaxSchemaDepth set max schema depth
func WithMaxSchemaDepth(max int) SchemaValidatorOption {
	return func(c *SchemaValidatorConfig) {
		if max >= 1 {
			c.MaxSchemaDepth = max
		}
	}
}

// WithMaxSchemaSize set max schema size
func WithMaxSchemaSize(max int) SchemaValidatorOption {
	return func(c *SchemaValidatorConfig) {
		if max >= 1 {
			c.MaxSchemaSize = max
		}
	}
}

// WithMaxTotalPropertiesKeysNum set max total properties keys num
func WithMaxTotalPropertiesKeysNum(max int) SchemaValidatorOption {
	return func(c *SchemaValidatorConfig) {
		if max >= 1 {
			c.MaxTotalPropertiesKeysNum = max
		}
	}
}

func (c *SchemaValidatorConfig) IsUltra() bool {
	return c.ValidateLevel == ValidateLevelUltra || c.ValidateLevel == ValidateLevelDefault
}

func (c *SchemaValidatorConfig) IsStrict() bool {
	return c.ValidateLevel == ValidateLevelStrict
}

func (c *SchemaValidatorConfig) IsLite() bool {
	return c.ValidateLevel == ValidateLevelLite
}

func (c *SchemaValidatorConfig) IsLoose() bool {
	return c.ValidateLevel == ValidateLevelLoose
}

// do not use in production
func (c *SchemaValidatorConfig) IsTest() bool {
	return c.ValidateLevel == ValidateLevelTest
}
