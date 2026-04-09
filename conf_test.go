package walle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatorConfigOptions(t *testing.T) {
	t.Run("Default configuration values", func(t *testing.T) {
		// Verify default values
		defaultConfig := DefaultValidatorConfig()
		assert.Equal(t, ValidateLevelDefault, defaultConfig.ValidateLevel)
		assert.True(t, defaultConfig.IsUltra())
		assert.False(t, defaultConfig.IsStrict())
		assert.Equal(t, 500, defaultConfig.MaxEnumItems)
		assert.Equal(t, 7500, defaultConfig.MaxEnumStringLength)
		assert.Equal(t, 250, defaultConfig.MaxEnumStringCheckThreshold)
		assert.Equal(t, 100, defaultConfig.MaxAnyOfItems)
		assert.Equal(t, 10, defaultConfig.MaxSchemaDepth)
		assert.Equal(t, 15000, defaultConfig.MaxSchemaSize)
		assert.Equal(t, 1000, defaultConfig.MaxTotalPropertiesKeysNum)
	})

	t.Run("WithMaxEnumItems option", func(t *testing.T) {
		config := DefaultValidatorConfig()
		// Apply option
		WithMaxEnumItems(100)(&config)
		// Verify only the targeted value changed
		assert.Equal(t, ValidateLevelDefault, config.ValidateLevel)
		assert.Equal(t, 100, config.MaxEnumItems)
		assert.Equal(t, 7500, config.MaxEnumStringLength)
		assert.Equal(t, 250, config.MaxEnumStringCheckThreshold)
		assert.Equal(t, 100, config.MaxAnyOfItems)
		assert.Equal(t, 10, config.MaxSchemaDepth)
		assert.Equal(t, 15000, config.MaxSchemaSize)
		assert.Equal(t, 1000, config.MaxTotalPropertiesKeysNum)
	})

	t.Run("WithMaxEnumStringLength option", func(t *testing.T) {
		config := DefaultValidatorConfig()
		// Apply option
		WithMaxEnumStringLength(5000)(&config)
		// Verify only the targeted value changed
		assert.Equal(t, ValidateLevelDefault, config.ValidateLevel)
		assert.Equal(t, 500, config.MaxEnumItems)
		assert.Equal(t, 5000, config.MaxEnumStringLength)
		assert.Equal(t, 250, config.MaxEnumStringCheckThreshold)
		assert.Equal(t, 100, config.MaxAnyOfItems)
		assert.Equal(t, 10, config.MaxSchemaDepth)
		assert.Equal(t, 15000, config.MaxSchemaSize)
		assert.Equal(t, 1000, config.MaxTotalPropertiesKeysNum)
	})

	t.Run("WithMaxEnumStringCheckThreshold option", func(t *testing.T) {
		config := DefaultValidatorConfig()
		// Apply option
		WithMaxEnumStringCheckThreshold(300)(&config)
		// Verify only the targeted value changed
		assert.Equal(t, ValidateLevelDefault, config.ValidateLevel)
		assert.Equal(t, 500, config.MaxEnumItems)
		assert.Equal(t, 7500, config.MaxEnumStringLength)
		assert.Equal(t, 300, config.MaxEnumStringCheckThreshold)
		assert.Equal(t, 100, config.MaxAnyOfItems)
		assert.Equal(t, 10, config.MaxSchemaDepth)
		assert.Equal(t, 15000, config.MaxSchemaSize)
		assert.Equal(t, 1000, config.MaxTotalPropertiesKeysNum)
	})

	t.Run("WithMaxAnyOfItems option", func(t *testing.T) {
		config := DefaultValidatorConfig()
		// Apply option
		WithMaxAnyOfItems(20)(&config)
		// Verify only the targeted value changed
		assert.Equal(t, ValidateLevelDefault, config.ValidateLevel)
		assert.Equal(t, 500, config.MaxEnumItems)
		assert.Equal(t, 7500, config.MaxEnumStringLength)
		assert.Equal(t, 250, config.MaxEnumStringCheckThreshold)
		assert.Equal(t, 20, config.MaxAnyOfItems)
		assert.Equal(t, 10, config.MaxSchemaDepth)
		assert.Equal(t, 15000, config.MaxSchemaSize)
		assert.Equal(t, 1000, config.MaxTotalPropertiesKeysNum)
	})

	t.Run("WithMaxSchemaDepth option", func(t *testing.T) {
		config := DefaultValidatorConfig()
		// Apply option
		WithMaxSchemaDepth(8)(&config)
		// Verify only the targeted value changed
		assert.Equal(t, ValidateLevelDefault, config.ValidateLevel)
		assert.Equal(t, 500, config.MaxEnumItems)
		assert.Equal(t, 7500, config.MaxEnumStringLength)
		assert.Equal(t, 250, config.MaxEnumStringCheckThreshold)
		assert.Equal(t, 100, config.MaxAnyOfItems)
		assert.Equal(t, 8, config.MaxSchemaDepth)
		assert.Equal(t, 15000, config.MaxSchemaSize)
		assert.Equal(t, 1000, config.MaxTotalPropertiesKeysNum)
	})

	t.Run("WithMaxSchemaSize option", func(t *testing.T) {
		config := DefaultValidatorConfig()
		// Apply option
		WithMaxSchemaSize(20000)(&config)
		// Verify only the targeted value changed
		assert.Equal(t, ValidateLevelDefault, config.ValidateLevel)
		assert.Equal(t, 500, config.MaxEnumItems)
		assert.Equal(t, 7500, config.MaxEnumStringLength)
		assert.Equal(t, 250, config.MaxEnumStringCheckThreshold)
		assert.Equal(t, 100, config.MaxAnyOfItems)
		assert.Equal(t, 10, config.MaxSchemaDepth)
		assert.Equal(t, 20000, config.MaxSchemaSize)
		assert.Equal(t, 1000, config.MaxTotalPropertiesKeysNum)
	})

	t.Run("WithMaxTotalPropertiesKeysNum option", func(t *testing.T) {
		config := DefaultValidatorConfig()
		// Apply option
		WithMaxTotalPropertiesKeysNum(1500)(&config)
		// Verify only the targeted value changed
		assert.Equal(t, ValidateLevelDefault, config.ValidateLevel)
		assert.Equal(t, 500, config.MaxEnumItems)
		assert.Equal(t, 7500, config.MaxEnumStringLength)
		assert.Equal(t, 250, config.MaxEnumStringCheckThreshold)
		assert.Equal(t, 100, config.MaxAnyOfItems)
		assert.Equal(t, 10, config.MaxSchemaDepth)
		assert.Equal(t, 15000, config.MaxSchemaSize)
		assert.Equal(t, 1500, config.MaxTotalPropertiesKeysNum)
	})

	t.Run("Multiple options together", func(t *testing.T) {
		// Create a config with multiple options applied
		config := DefaultValidatorConfig()

		// Apply multiple options
		options := []SchemaValidatorOption{
			WithMaxEnumItems(200),
			WithMaxEnumStringLength(3000),
			WithMaxAnyOfItems(15),
			WithMaxSchemaDepth(7),
		}

		for _, option := range options {
			option(&config)
		}

		// Verify all changes were applied correctly
		assert.Equal(t, ValidateLevelDefault, config.ValidateLevel)
		assert.Equal(t, 200, config.MaxEnumItems)
		assert.Equal(t, 3000, config.MaxEnumStringLength)
		assert.Equal(t, 250, config.MaxEnumStringCheckThreshold)
		assert.Equal(t, 15, config.MaxAnyOfItems)
		assert.Equal(t, 7, config.MaxSchemaDepth)
		assert.Equal(t, 15000, config.MaxSchemaSize)
		assert.Equal(t, 1000, config.MaxTotalPropertiesKeysNum)
	})

	t.Run("WithValidateLevel lite", func(t *testing.T) {
		config := DefaultValidatorConfig()
		WithValidateLevel(ValidateLevelLite)(&config)
		assert.Equal(t, ValidateLevelLite, config.ValidateLevel)
	})

	t.Run("Multiple options with ValidateLevel", func(t *testing.T) {
		config := DefaultValidatorConfig()

		// Apply multiple options including ValidateLevel
		options := []SchemaValidatorOption{
			WithValidateLevel(ValidateLevelUltra),
			WithMaxEnumItems(200),
			WithMaxSchemaDepth(7),
		}

		for _, option := range options {
			option(&config)
		}

		// Verify all changes were applied correctly
		assert.Equal(t, ValidateLevelUltra, config.ValidateLevel)
		assert.Equal(t, 200, config.MaxEnumItems)
		assert.Equal(t, 7500, config.MaxEnumStringLength)
		assert.Equal(t, 250, config.MaxEnumStringCheckThreshold)
		assert.Equal(t, 100, config.MaxAnyOfItems)
		assert.Equal(t, 7, config.MaxSchemaDepth)
		assert.Equal(t, 15000, config.MaxSchemaSize)
		assert.Equal(t, 1000, config.MaxTotalPropertiesKeysNum)
	})
}

func TestIsGreaterThanStrict(t *testing.T) {
	cases := []struct {
		level ValidateLevel
		want  bool
	}{
		{ValidateLevelLoose, false},
		{ValidateLevelLite, false},
		{ValidateLevelStrict, true},
		{ValidateLevelUltra, true},
		{ValidateLevelDefault, true},
		{ValidateLevelTest, true},
	}
	for _, tc := range cases {
		t.Run(string(tc.level), func(t *testing.T) {
			c := DefaultValidatorConfig()
			WithValidateLevel(tc.level)(&c)
			assert.Equal(t, tc.want, c.IsGreaterThanStrict())
		})
	}
}
