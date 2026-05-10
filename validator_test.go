package walle

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const validatorJSONLMaxLine = 1024 * 1024

func newJSONLScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), validatorJSONLMaxLine)
	return sc
}

func jsonlScanErr(testName, file string, err error) string {
	if errors.Is(err, bufio.ErrTooLong) {
		return fmt.Sprintf("%s: %s: line longer than %d bytes; split the line or increase validatorJSONLMaxLine",
			testName, file, validatorJSONLMaxLine)
	}
	return fmt.Sprintf("%s: read %s: %v", testName, file, err)
}

func loadValidatorCasePair(t *testing.T, testName string) (
	valid []string,
	invalid []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	},
) {
	t.Helper()
	base := filepath.Join("testdata", "validator_cases", testName)

	vf, err := os.Open(filepath.Join(base, "valid.jsonl"))
	if err != nil {
		t.Fatalf("%s: open valid.jsonl: %v", testName, err)
	}
	defer vf.Close()
	sc := newJSONLScanner(vf)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		valid = append(valid, line)
	}
	if err = sc.Err(); err != nil {
		t.Fatal(jsonlScanErr(testName, "valid.jsonl", err))
	}

	invf, err := os.Open(filepath.Join(base, "invalid.jsonl"))
	if err != nil {
		t.Fatalf("%s: open invalid.jsonl: %v", testName, err)
	}
	defer invf.Close()
	sc2 := newJSONLScanner(invf)
	for sc2.Scan() {
		line := strings.TrimSpace(sc2.Text())
		if line == "" {
			continue
		}
		var row struct {
			Schema    string `json:"schema"`
			Expect    string `json:"expect"`
			Unmarshal bool   `json:"unmarshal"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			prefix := line
			if len(prefix) > 200 {
				prefix = prefix[:200]
			}
			t.Fatalf("%s: invalid.jsonl line: %v\n%s", testName, err, prefix)
		}
		invalid = append(invalid, struct {
			schema         string
			expectedErr    string
			isUnmarshalErr bool
		}{row.Schema, row.Expect, row.Unmarshal})
	}
	if err := sc2.Err(); err != nil {
		t.Fatal(jsonlScanErr(testName, "invalid.jsonl", err))
	}
	return valid, invalid
}

// validatorJSONLSuiteDirs matches testdata/validator_cases/<name>/; add a directory and entry here for new suites.
var validatorJSONLSuiteDirs = []string{
	"TestBasicTypes",
	"TestSingleTypeInArray",
	"TestAdditionalProperties",
	"TestRequired",
	"TestKeywordsValidation",
	"TestReferences",
	"TestAnyOf",
	"TestDefs",
	"TestNumberFormat",
	"TestRefInProperties",
	"TestTypeLocation",
	"TestNestedDefsDepth",
	"TestRangeConstraints",
	"TestID",
	"TestDescription",
	"TestEnforcerCases",
}

func TestValidatorJSONLSuites(t *testing.T) {
	for _, dir := range validatorJSONLSuiteDirs {
		t.Run(dir, func(t *testing.T) {
			v, inv := loadValidatorCasePair(t, dir)
			runTestCases(t, v, inv)
		})
	}
}

func TestMaxTotalProperties(t *testing.T) {
	validator := newSchemaValidator()

	// Create a schema with too many properties
	properties1 := make(SchemaDict)
	properties2 := make(SchemaDict)
	for i := 1; i <= 10000; i++ {
		properties1[fmt.Sprintf("%d", i)] = SchemaDict{"type": "string"}
		properties2[fmt.Sprintf("k%d", i)] = SchemaDict{"type": "string"}
	}
	properties2["k10001"] = SchemaDict{"type": "string"}

	// schema1 := SchemaDict{
	// 	"type":       "object",
	// 	"properties": properties1,
	// }
	schema2 := SchemaDict{
		"type":       "object",
		"properties": properties2,
	}

	// TODO: MaxSchemaSize maybe too small
	// if err1 := validator.Validate(schema1); err1 != nil {
	// 	t.Errorf("Valid schema failed: %v", err1)
	// }

	if err2 := validator.Validate(schema2); err2 == nil {
		t.Errorf("schema with too many properties should have failed")
	} else {
		expectedErr1 := "total number of properties keys across all objects exceeds maximum"
		expectedErr2 := "schema exceeds maximum allowed size"
		errMsg := strings.ToLower(err2.Error())
		if !strings.Contains(errMsg, strings.ToLower(expectedErr1)) && !strings.Contains(errMsg, strings.ToLower(expectedErr2)) {
			t.Errorf("Expected error containing '%s' or '%s', got '%s'", expectedErr1, expectedErr2, err2.Error())
		}
	}
}

func TestEnumStringLength(t *testing.T) {
	createEnumJSON := func(prefix string, count int) string {
		var values []string
		for i := 0; i < count; i++ {
			values = append(values, fmt.Sprintf(`"%s%d"`, prefix, i))
		}
		return fmt.Sprintf(`{"type": "string", "enum": [%s]}`, strings.Join(values, ", "))
	}

	createNumericEnumJSON := func(count int) string {
		var values []string
		for i := 0; i < count; i++ {
			values = append(values, fmt.Sprintf("%d", i))
		}
		return fmt.Sprintf(`{"type": "number", "enum": [%s]}`, strings.Join(values, ", "))
	}

	createLargeNumericEnumJSON := func(value float64, count int) string {
		var values []string
		for i := 0; i < count; i++ {
			values = append(values, fmt.Sprintf("%f", value))
		}
		return fmt.Sprintf(`{"type": "number", "enum": [%s]}`, strings.Join(values, ", "))
	}

	createLargeIntegerEnumJSON := func(value int64, count int) string {
		var values []string
		for i := 0; i < count; i++ {
			values = append(values, fmt.Sprintf("%d", value))
		}
		return fmt.Sprintf(`{"type": "integer", "enum": [%s]}`, strings.Join(values, ", "))
	}

	validCases := []string{
		// Less than 250 enum values
		createEnumJSON("long_value_", 249),
		// Exactly 250 enum values
		createEnumJSON("long_value_", 250),
		// More than 250 but short string
		createEnumJSON("s_", 300),
		// Numeric enum values
		createNumericEnumJSON(250),
		// Integer enum values
		createNumericEnumJSON(300),
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// More than 250 values and long string
		{
			createEnumJSON("very_loooooooooooong_enum_value_", 251),
			"total string length of enum values",
			false,
		},
		// Numeric type but value is too large
		{
			createLargeNumericEnumJSON(123456789010.123456789, 499),
			"total string length of enum values",
			false,
		},
		// Integer type but value is too large
		{
			createLargeIntegerEnumJSON((1<<53)-1, 499),
			"total string length of enum values",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestConcurrentValidation(t *testing.T) {
	// Test concurrent validation
	concurrentNum := 32
	t.Run("Concurrent validation", func(t *testing.T) {
		schema := `{"type":"object","required":["id","name","details","tags","metadata"],"properties":{"id":{"type":"string","minLength":5,"maxLength":50},"name":{"type":"string","minLength":3,"maxLength":100},"age":{"type":"integer","minimum":0,"maximum":150},"email":{"type":"string"},"details":{"type":"object","required":["description","status"],"properties":{"description":{"type":"string"},"status":{"type":"string","enum":["active","inactive","pending"]},"createdAt":{"type":"string"},"score":{"type":"number","minimum":0,"maximum":10}}},"tags":{"type":"array","minItems":1,"maxItems":10,"items":{"type":"string","minLength":2}},"metadata":{"type":"object","additionalProperties":{"type":"string"}},"settings":{"type":"object","properties":{"notifications":{"type":"boolean"},"theme":{"type":"string","enum":["light","dark","system"]},"fontSize":{"type":"integer","minimum":8,"maximum":24}}}},"additionalProperties":false}`

		// Channel to collect execution times
		timings := make(chan time.Duration, concurrentNum)
		var wg sync.WaitGroup
		wg.Add(concurrentNum)

		for i := 0; i < concurrentNum; i++ {
			go func() {
				defer wg.Done()

				startTime := time.Now()

				validator := newSchemaValidator(WithValidateLevel(ValidateLevelTest))
				if err := validator.Validate(schema); err != nil {
					t.Errorf("Concurrent validation failed: %v", err)
				}

				executionTime := time.Since(startTime)
				timings <- executionTime
			}()
		}

		go func() {
			wg.Wait()
			close(timings)
		}()

		var times []time.Duration
		for duration := range timings {
			times = append(times, duration)
		}

		// Calculate statistics
		var totalTime time.Duration
		minTime := times[0]
		maxTime := times[0]

		for _, duration := range times {
			totalTime += duration
			if duration < minTime {
				minTime = duration
			}
			if duration > maxTime {
				maxTime = duration
			}
		}

		avgTime := totalTime / time.Duration(len(times))

		// Print statistics
		t.Logf("Validation Performance Statistics:")
		t.Logf("  Total goroutines: %d", concurrentNum)
		t.Logf("  Average time: %v", avgTime)
		t.Logf("  Minimum time: %v", minTime)
		t.Logf("  Maximum time: %v", maxTime)
	})

}

func TestLargeSchemaHandling(t *testing.T) {
	// Test large schema handling
	t.Run("Large schema handling", func(t *testing.T) {
		validator := newSchemaValidator()
		// Create a large schema with many properties
		properties := make([]string, 10000)
		for i := 0; i < 10000; i++ {
			properties[i] = fmt.Sprintf(`"prop%d": {"type": "string"}`, i)
		}

		largeSchema := fmt.Sprintf(`{
			"type": "object",
			"properties": {
				%s
			}
		}`, strings.Join(properties, ",\n"))

		err := validator.Validate(largeSchema)
		if err == nil {
			t.Error("Expected error for large schema")
		} else {
			errLower := strings.ToLower(err.Error())
			if !strings.Contains(errLower, "schema exceeds maximum allowed size") &&
				!strings.Contains(errLower, "exceeds maximum") {
				t.Errorf("Expected error about schema size, got: %v", err)
			}
		}
	})
}
func TestValidateAPI(t *testing.T) {
	t.Run("Validate API", func(t *testing.T) {
		must := require.New(t)
		validator := newSchemaValidator()
		err := validator.Validate(make(map[string]struct{}))
		must.Error(err)
		must.Contains(err.Error(), "input schema must be a string or map")
	})
}

func TestSchemaValidatorWithCustomConfig(t *testing.T) {
	t.Run("Custom configuration through options", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with custom options
		validator := newSchemaValidator(
			WithMaxEnumItems(250),
			WithMaxSchemaDepth(10),
			WithMaxSchemaSize(30000),
		)

		// Check if the config values were applied correctly
		must.Equal(250, validator.config.MaxEnumItems)
		must.Equal(10, validator.config.MaxSchemaDepth)
		must.Equal(30000, validator.config.MaxSchemaSize)

		// Default values for others
		must.Equal(7500, validator.config.MaxEnumStringLength)
		must.Equal(250, validator.config.MaxEnumStringCheckThreshold)
		must.Equal(100, validator.config.MaxAnyOfItems)
		must.Equal(1000, validator.config.MaxTotalPropertiesKeysNum)
	})

	t.Run("MaxEnumStringLength and MaxEnumStringCheckThreshold limit", func(t *testing.T) {
		must := require.New(t)
		validator := newSchemaValidator(
			WithMaxEnumStringCheckThreshold(3),
			WithMaxEnumStringLength(50),
		)

		// Create a schema with multiple enum values and total length exceeding the limit
		longEnumSchema := `{
			"type": "string",
			"enum": [
				"string_value_1",
				"string_value_2", 
				"string_value_3",
				"string_value_4",
				"string_value_5"
			]
    	}`

		// Should fail because total length exceeds the limit
		err := validator.Validate(longEnumSchema)
		must.Error(err)
		must.Contains(err.Error(), "exceeds maximum limit of 50 characters when enum has more than 3 values")

		// Now increase the total length limit but keep the threshold unchanged
		validator = newSchemaValidator(
			WithMaxEnumStringCheckThreshold(3),
			WithMaxEnumStringLength(100),
		)
		err = validator.Validate(longEnumSchema)
		must.NoError(err)

		// Another test: keep low limit but raise threshold to avoid triggering check
		validator = newSchemaValidator(
			WithMaxEnumStringCheckThreshold(10),
			WithMaxEnumStringLength(50),
		)
		err = validator.Validate(longEnumSchema)
		must.NoError(err)
	})

	t.Run("MaxEnumItems limit", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with a low enum item limit
		validator := newSchemaValidator(
			WithMaxEnumItems(3),
		)

		// Schema with more enum items than allowed
		schemaWithLargeEnum := `{
			"type": "string",
			"enum": ["option1", "option2", "option3", "option4", "option5"]
		}`

		// This validation should fail due to too many enum items
		err := validator.Validate(schemaWithLargeEnum)
		must.Error(err)
		must.Contains(err.Error(), "enum array cannot have more than 3 items")

		// Now increase the enum limit and try again
		validator = newSchemaValidator(
			WithMaxEnumItems(5),
		)

		// This should now succeed
		err = validator.Validate(schemaWithLargeEnum)
		must.NoError(err)
	})

	t.Run("MaxAnyOfItems limit", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with a low anyOf item limit
		validator := newSchemaValidator(
			WithMaxAnyOfItems(2),
		)

		// Schema with more anyOf items than allowed
		schemaWithManyAnyOf := `{
			"anyOf": [
				{"type": "string"},
				{"type": "number"},
				{"type": "boolean"}
			]
		}`

		// This validation should fail due to too many anyOf items
		err := validator.Validate(schemaWithManyAnyOf)
		must.Error(err)
		must.Contains(err.Error(), "anyOf must have 1-2 items")

		// Now increase the limit and try again
		validator = newSchemaValidator(
			WithMaxAnyOfItems(3),
		)
		err = validator.Validate(schemaWithManyAnyOf)
		must.NoError(err)
	})

	t.Run("MaxSchemaDepth limit", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with custom depth limit
		validator := newSchemaValidator(
			WithMaxSchemaDepth(3),
		)

		// Create a deeply nested schema that should exceed the depth limit
		deepSchema := `{
			"type": "object",
			"properties": {
				"level1": {
					"type": "object",
					"properties": {
						"level2": {
							"type": "object",
							"properties": {
								"level3": {
									"type": "object",
									"properties": {
										"level4": {
											"type": "string"
										}
									}
								}
							}
						}
					}
				}
			}
		}`

		// This validation should fail due to depth exceeding the limit
		err := validator.Validate(deepSchema)
		must.Error(err)
		must.Contains(err.Error(), "schema depth exceeds maximum limit of 3")

		// Now increase the depth limit and try again
		validator = newSchemaValidator(
			WithMaxSchemaDepth(4),
		)
		err = validator.Validate(deepSchema)
		must.NoError(err)
	})

	t.Run("MaxSchemaSize limit", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with a very low schema size limit
		validator := newSchemaValidator(
			WithMaxSchemaSize(100), // Very small limit
		)

		// Create a schema that exceeds this small size limit
		largeSchema := `{
			"type": "object",
			"properties": {
				"prop1": {"type": "string", "description": "This is property 1 with a somewhat lengthy description"},
				"prop2": {"type": "number", "description": "This is property 2 with another lengthy description"},
				"prop3": {"type": "boolean", "description": "And here we have property 3 with yet another description"}
			},
			"required": ["prop1", "prop2"]
		}`

		// This validation should fail due to size exceeding the limit
		err := validator.Validate(largeSchema)
		must.Error(err)
		must.Contains(err.Error(), "schema exceeds maximum allowed size")

		// Now increase the size limit and try again
		validator = newSchemaValidator(
			WithMaxSchemaSize(1000),
		)
		err = validator.Validate(largeSchema)
		must.NoError(err)
	})

	t.Run("MaxTotalPropertiesKeysNum limit", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with a low property keys limit
		validator := newSchemaValidator(
			WithMaxTotalPropertiesKeysNum(5),
		)

		// Create a schema with more properties than the limit
		schemaWithManyProperties := `{
			"type": "object",
			"properties": {
				"prop1": {"type": "string"},
				"prop2": {"type": "number"},
				"prop3": {"type": "boolean"},
				"prop4": {"type": "array", "items": {"type": "string"}},
				"prop5": {"type": "object", "properties": {"subprop": {"type": "string"}}},
				"prop6": {"type": "integer"}
			}
		}`

		// This validation should fail due to too many properties
		err := validator.Validate(schemaWithManyProperties)
		must.Error(err)
		must.Contains(err.Error(), "total number of properties keys(6) across all objects exceeds maximum limit of 5")

		// Now increase the limit and try again
		validator = newSchemaValidator(
			WithMaxTotalPropertiesKeysNum(7), // 6 + 1
		)
		err = validator.Validate(schemaWithManyProperties)
		must.NoError(err)
	})
}

func runTestCases(t *testing.T, validCases []string, invalidCases []struct {
	schema         string
	expectedErr    string
	isUnmarshalErr bool
}) {
	must := require.New(t)
	validator := newSchemaValidator(
		WithValidateLevel(ValidateLevelTest),
		WithMaxSchemaDepth(5),
	)

	for _, schema := range validCases {
		err := validator.Validate(schema)
		must.NoError(err, "Valid schema failed: %v\nSchema: %v", err, schema)
	}

	for _, tc := range invalidCases {
		err := validator.Validate(tc.schema)
		must.Error(err, "invalid schema should have failed: %s\nSchema: %v", tc.expectedErr, tc.schema)

		if err != nil {
			if tc.isUnmarshalErr {
				must.True(IsUnmarshalError(err),
					"Expected UnmarshalError, got: %T\nSchema: %v", err, tc.schema)
			} else {
				must.True(IsSchemaError(err),
					"Expected SchemaError, got: %T\nSchema: %v", err, tc.schema)
			}

			errLower := strings.ToLower(err.Error())
			expectedLower := strings.ToLower(tc.expectedErr)
			must.Contains(errLower, expectedLower,
				"Expected error containing '%s', got '%s', %v", tc.expectedErr, err.Error(), tc.schema)
		}
	}
}

func TestUltraValidate(t *testing.T) {
	must := require.New(t)
	validator := newSchemaValidator(WithValidateLevel(ValidateLevelUltra))

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// required contains duplicate items
		{
			`{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": ["name", "name"]
			}`,
			"duplicate items in required array",
			false,
		},
		// duplicate types in type array
		{
			`{
				"type": ["string", "string"]
			}`,
			"duplicate types in type array",
			false,
		},
	}

	for _, tc := range invalidCases {
		err := validator.Validate(tc.schema)
		must.Error(err, "invalid schema should have failed: %s\nSchema: %v", tc.expectedErr, tc.schema)

		if err != nil {
			if tc.isUnmarshalErr {
				must.True(IsUnmarshalError(err),
					"Expected UnmarshalError, got: %T\nSchema: %v", err, tc.schema)
			} else {
				must.True(IsSchemaError(err),
					"Expected SchemaError, got: %T\nSchema: %v", err, tc.schema)
			}

			errLower := strings.ToLower(err.Error())
			expectedLower := strings.ToLower(tc.expectedErr)
			must.Contains(errLower, expectedLower,
				"Expected error containing '%s', got '%s', %v", tc.expectedErr, err.Error(), tc.schema)
		}
	}
}

func TestUltraTypeArrayKeywordValidationIsOrderIndependent(t *testing.T) {
	must := require.New(t)
	validator := newSchemaValidator(WithValidateLevel(ValidateLevelUltra))

	schemas := []string{
		`{"type":["integer","string"],"enum":[1,"a"],"minimum":0}`,
		`{"type":["string","integer"],"enum":[1,"a"],"minimum":0}`,
	}

	for _, schema := range schemas {
		err := validator.Validate(schema)
		must.Error(err)
		must.Contains(strings.ToLower(err.Error()), "invalid keywords: minimum")
	}
}

func TestUltraTypeArrayKeywordCheckPrecedesRangeValidation(t *testing.T) {
	must := require.New(t)
	validator := newSchemaValidator(WithValidateLevel(ValidateLevelUltra))

	schema := `{"type":["string","integer"],"enum":[1,"a"],"minimum":"x"}`
	err := validator.Validate(schema)
	must.Error(err)
	must.Contains(strings.ToLower(err.Error()), "invalid keywords: minimum")
}

func TestLiteAllowsMultipleTypesWithItems(t *testing.T) {
	must := require.New(t)
	schema := `{"additionalProperties": false, "properties": {"files": {"description": "List of files to read; request related files together when allowed", "items": {"additionalProperties": false, "properties": {"line_ranges": {"description": "Optional line ranges to read. Each range is a [start, end] tuple with 1-based inclusive line numbers. Use multiple ranges for non-contiguous sections.", "items": {"items": {"type": "integer"}, "maxItems": 2, "minItems": 2, "type": "array"}, "type": ["array", "null"]}, "path": {"description": "Path to the file to read, relative to the workspace", "type": "string"}}, "required": ["path", "line_ranges"], "type": "object"}, "minItems": 1, "type": "array"}}, "required": ["files"], "type": "object"}`

	lite := newSchemaValidator(WithValidateLevel(ValidateLevelLite))
	must.NoError(lite.Validate(schema), "lite should accept type[] + items")

	for _, level := range []ValidateLevel{ValidateLevelStrict, ValidateLevelUltra, ValidateLevelDefault} {
		v := newSchemaValidator(WithValidateLevel(level))
		err := v.Validate(schema)
		must.Error(err, "level %s should reject", level)
		must.Contains(strings.ToLower(err.Error()), "multiple types")
	}
}

func TestLiteAllowsReservedPropertyNamesInProperties(t *testing.T) {
	must := require.New(t)
	// properties 下存在名为 required 的字段（与 required 关键字同名）
	schema := `{
		"type": "object",
		"properties": {
			"required": { "type": "string", "description": "not the keyword" }
		},
		"additionalProperties": false
	}`

	lite := newSchemaValidator(WithValidateLevel(ValidateLevelLite))
	must.NoError(lite.Validate(schema), "lite should allow reserved-looking property names")

	loose := newSchemaValidator(WithValidateLevel(ValidateLevelLoose))
	must.NoError(loose.Validate(schema), "loose should allow reserved-looking property names")

	ultra := newSchemaValidator(WithValidateLevel(ValidateLevelUltra))
	err := ultra.Validate(schema)
	must.Error(err, "ultra should reject reserved property names")
	must.Contains(strings.ToLower(err.Error()), "reserved")
}
