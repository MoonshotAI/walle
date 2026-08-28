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
			createEnumJSON("very_loooooooooooong_enum_value_", 1001),
			"enum array cannot have more than 1000 items",
			false,
		},
		// Numeric type but value is too large
		{
			createLargeNumericEnumJSON(123456789010.123456789, 1001),
			"enum array cannot have more than 1000 items",
			false,
		},
		// Integer type but value is too large
		{
			createLargeIntegerEnumJSON((1<<53)-1, 1001),
			"enum array cannot have more than 1000 items",
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
		must.Equal(75000, validator.config.MaxEnumStringLength)
		must.Equal(2500, validator.config.MaxEnumStringCheckThreshold)
		must.Equal(500, validator.config.MaxAnyOfItems)
		must.Equal(3000, validator.config.MaxTotalPropertiesKeysNum)
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

func TestUltraUnsupportedKeywordsErrorOrderIsStable(t *testing.T) {
	must := require.New(t)
	validator := newSchemaValidator(WithValidateLevel(ValidateLevelUltra))
	schema := `{"type":"object","zzz":1,"aaa":2}`

	for i := 0; i < 30; i++ {
		err := validator.Validate(schema)
		must.Error(err)
		must.Contains(strings.ToLower(err.Error()), "unsupported keywords: aaa, zzz")
	}
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

// Termination is about whether a *finite* instance exists, not about whether the
// schema mentions itself. A cycle only makes the schema unsatisfiable when every
// hop is mandatory: a required object property always forces one more level,
// while an array without a positive minItems can stop at the empty array.
//
// This used to be masked twice over: PostValidateRefs skipped the whole check
// whenever the root happened to terminate, and CheckRefTermination treated the
// required properties of an object as "any one of them terminates" instead of
// "all of them must".
func TestRefTerminationDistinguishesFiniteFromInfiniteRecursion(t *testing.T) {
	cases := []struct {
		name string
		// witness is a JSON instance proving satisfiability, or why none exists
		witness string
		schema  string
		reject  bool
	}{
		{
			name:    "optional self reference stops immediately",
			witness: `{"node":{}}`,
			schema:  `{"type":"object","properties":{"node":{"$ref":"#/$defs/N"}},"$defs":{"N":{"type":"object","properties":{"next":{"$ref":"#/$defs/N"}}}}}`,
		},
		{
			name:    "required array self reference stops at the empty array",
			witness: `{"children":[]}`,
			schema:  `{"type":"object","properties":{"children":{"type":"array","items":{"$ref":"#"}}},"required":["children"]}`,
		},
		{
			name:    "mutual recursion through an unbounded array stops at the empty array",
			witness: `{"root":{"level2":[]}}`,
			schema:  `{"type":"object","properties":{"root":{"$ref":"#/$defs/L1"}},"required":["root"],"$defs":{"L1":{"type":"object","properties":{"level2":{"$ref":"#/$defs/L2"}},"required":["level2"]},"L2":{"type":"array","items":{"$ref":"#/$defs/L1"}}}}`,
		},
		{
			name:    "required self reference never bottoms out",
			witness: "none: every instance needs one more 'next'",
			schema:  `{"type":"object","properties":{"node":{"$ref":"#/$defs/N"}},"$defs":{"N":{"type":"object","properties":{"next":{"$ref":"#/$defs/N"}},"required":["next"]}}}`,
			reject:  true,
		},
		{
			name:    "mutual recursion where every hop is required",
			witness: "none: A needs B, B needs A",
			schema:  `{"type":"object","properties":{"root":{"$ref":"#/$defs/A"}},"required":["root"],"$defs":{"A":{"type":"object","properties":{"b":{"$ref":"#/$defs/B"}},"required":["b"]},"B":{"type":"object","properties":{"a":{"$ref":"#/$defs/A"}},"required":["a"]}}}`,
			reject:  true,
		},
		{
			name:    "minItems forces the array cycle to continue",
			witness: "none: the array can never be empty",
			schema:  `{"type":"object","properties":{"root":{"$ref":"#/$defs/L1"}},"required":["root"],"$defs":{"L1":{"type":"object","properties":{"level2":{"$ref":"#/$defs/L2"}},"required":["level2"]},"L2":{"type":"array","minItems":1,"items":{"$ref":"#/$defs/L1"}}}}`,
			reject:  true,
		},
		{
			name:    "non-terminating branch hidden behind a terminating sibling",
			witness: "none: 'loop' is required alongside the harmless 'label'",
			schema:  `{"type":"object","properties":{"label":{"type":"string"},"loop":{"$ref":"#/$defs/N"}},"required":["label","loop"],"$defs":{"N":{"type":"object","properties":{"next":{"$ref":"#/$defs/N"}},"required":["next"]}}}`,
			reject:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := newSchemaValidator(WithValidateLevel(ValidateLevelLite)).Validate(tc.schema)
			if tc.reject {
				if err == nil {
					t.Fatalf("expected rejection (%s)", tc.witness)
				}
				if !strings.Contains(err.Error(), "infinite recursion") {
					t.Fatalf("expected an infinite recursion error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected schema to pass, %s is a valid instance: %v", tc.witness, err)
			}
		})
	}
}

// A lower bound above its upper bound cannot be satisfied by any instance. lite
// keeps accepting such schemas so that existing callers do not start failing,
// but strict and above reject them, and Canonical degrades the offending
// subschema to {} rather than deleting the bounds -- deleting them would turn
// "impossible" into "anything goes".
func TestBoundConflictsRejectedFromStrictUpwards(t *testing.T) {
	cases := []struct {
		name   string
		schema string
	}{
		{
			name:   "minLength above maxLength",
			schema: `{"type":"object","properties":{"a":{"type":"string","minLength":10,"maxLength":2}}}`,
		},
		{
			name:   "minimum above maximum",
			schema: `{"type":"object","properties":{"a":{"type":"integer","minimum":10,"maximum":2}}}`,
		},
		{
			name:   "minItems above maxItems",
			schema: `{"type":"object","properties":{"a":{"type":"array","minItems":10,"maxItems":2,"items":{"type":"string"}}}}`,
		},
	}

	accepting := []ValidateLevel{ValidateLevelLoose, ValidateLevelLite}
	rejecting := []ValidateLevel{ValidateLevelStrict, ValidateLevelUltra}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, level := range accepting {
				if err := newSchemaValidator(WithValidateLevel(level)).Validate(tc.schema); err != nil {
					t.Errorf("%s should still accept the schema, got %v", level, err)
				}
			}
			for _, level := range rejecting {
				err := newSchemaValidator(WithValidateLevel(level)).Validate(tc.schema)
				if err == nil {
					t.Errorf("%s should reject the schema", level)
					continue
				}
				if !strings.Contains(err.Error(), "cannot be greater than") {
					t.Errorf("%s: expected a bound conflict error, got %v", level, err)
				}
			}

			schema, err := ParseSchema(tc.schema)
			if err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}
			result, _ := schema.Canonical()
			if !strings.Contains(result, `"a":{}`) {
				t.Errorf("expected the property to degrade to {}, got %s", result)
			}
		})
	}
}

// Bounds that make sense together must survive untouched at every level.
func TestConsistentBoundsAreUntouched(t *testing.T) {
	const schema = `{"type":"object","properties":{"a":{"type":"string","minLength":2,"maxLength":10}}}`

	for _, level := range []ValidateLevel{ValidateLevelLoose, ValidateLevelLite, ValidateLevelStrict, ValidateLevelUltra} {
		if err := newSchemaValidator(WithValidateLevel(level)).Validate(schema); err != nil {
			t.Errorf("%s should accept consistent bounds, got %v", level, err)
		}
	}

	parsed, err := ParseSchema(schema)
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	result, warnErr := parsed.Canonical()
	if warnErr != nil {
		t.Fatalf("expected no warning, got %v", warnErr)
	}
	for _, frag := range []string{`"minLength":2`, `"maxLength":10`} {
		if !strings.Contains(result, frag) {
			t.Errorf("expected %s to survive, got %s", frag, result)
		}
	}
}

// The node that started all of this, taken from automation_update.parameters.json.
// The use site restates the definition's type and minLength, adds a description
// and an unsupported format, and points at the definition with $ref. lite used
// to answer 400 because a keyword appeared on both sides at all.
func TestReportedRefSiblingSchemaIsAccepted(t *testing.T) {
	const schema = `{
		"type": "object",
		"properties": {
			"__schema20": {
				"type": "string",
				"minLength": 1,
				"format": "uuid",
				"description": "Target thread UUID for heartbeat automations. Prefer destination=thread for the current local thread instead of inventing or copying raw thread ids.",
				"$ref": "#/$defs/__schema2"
			}
		},
		"$defs": {
			"__schema2": { "type": "string", "minLength": 1 }
		}
	}`

	for _, level := range []ValidateLevel{ValidateLevelLoose, ValidateLevelLite, ValidateLevelStrict} {
		if err := newSchemaValidator(WithValidateLevel(level)).Validate(schema); err != nil {
			t.Errorf("%s must accept the reported node, got %v", level, err)
		}
	}

	if err := newSchemaValidator(WithValidateLevel(ValidateLevelUltra)).Validate(schema); err == nil {
		t.Fatal("ultra must still report the unsupported format")
	} else if !strings.Contains(strings.ToLower(err.Error()), "unsupported keywords") {
		t.Fatalf("expected an unsupported-keyword error, got %v", err)
	}

	parsed, err := ParseSchema(schema)
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	result, warnErr := parsed.Canonical()
	if warnErr == nil || !strings.Contains(strings.ToLower(warnErr.Error()), "format") {
		t.Fatalf("expected a warning about format, got %v", warnErr)
	}

	for _, want := range []string{
		`"type":"string"`,
		`"minLength":1`,
		`"description":"Target thread UUID for heartbeat automations. Prefer destination=thread for the current local thread instead of inventing or copying raw thread ids."`,
	} {
		if !strings.Contains(result, want) {
			t.Errorf("expected %s to survive, got %s", want, result)
		}
	}
	if strings.Contains(result, `"format"`) {
		t.Errorf("format is unsupported and must be dropped, got %s", result)
	}
}

// Constraining an instance both directly and through anyOf is a legal conjunction
// under 2020-12, and so is requiring a property the schema does not describe or
// listing an enum value the type rules out. None of them can be handed to the
// enforcer as written, so only the canonicalising levels report them; lite has to
// accept the schema and leave the rewriting to Canonical.
func TestLiteAcceptsStructuralShapesCanonicalCanRewrite(t *testing.T) {
	cases := []struct {
		name string
		in   string
		// want is the complete canonical form, so a rewrite that quietly loosens
		// the schema cannot slip past
		want string
	}{
		{
			name: "type beside anyOf is pushed into the branches",
			in:   `{"type":"object","properties":{"v":{"type":"string","anyOf":[{"minLength":1},{"maxLength":9}]}}}`,
			want: `{"properties":{"v":{"anyOf":[{"minLength":1,"type":"string"},{"maxLength":9,"type":"string"}]}},"type":"object"}`,
		},
		{
			name: "a branch the parent type rules out is dropped",
			in:   `{"type":"object","properties":{"v":{"type":"string","anyOf":[{"minLength":1},{"type":"integer"}]}}}`,
			want: `{"properties":{"v":{"anyOf":[{"minLength":1,"type":"string"}]}},"type":"object"}`,
		},
		{
			name: "every branch ruled out leaves nothing to satisfy",
			in:   `{"type":"object","properties":{"v":{"type":"string","anyOf":[{"type":"integer"},{"type":"boolean"}]}}}`,
			want: `{"properties":{"v":{}},"type":"object"}`,
		},
		{
			name: "a contradiction behind a branch's $ref also empties the node",
			in:   `{"type":"object","properties":{"v":{"type":"string","anyOf":[{"$ref":"#/$defs/S"}]}},"$defs":{"S":{"type":"number"}}}`,
			want: `{"$defs":{"S":{"type":"number"}},"properties":{"v":{}},"type":"object"}`,
		},
		{
			name: "the stricter of the two bounds survives distribution",
			in:   `{"type":"object","properties":{"v":{"minLength":20,"anyOf":[{"type":"string","minLength":10}]}}}`,
			want: `{"properties":{"v":{"anyOf":[{"minLength":20,"type":"string"}]}},"type":"object"}`,
		},
		{
			name: "an annotation beside anyOf stays where it is",
			in:   `{"type":"object","properties":{"v":{"description":"x","anyOf":[{"type":"string"},{"type":"integer"}]}}}`,
			want: `{"properties":{"v":{"anyOf":[{"type":"string"},{"type":"integer"}],"description":"x"}},"type":"object"}`,
		},
		{
			name: "an undeclared required entry is pruned and the declared one kept",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"required":["a","b"]}`,
			want: `{"properties":{"a":{"type":"string"}},"required":["a"],"type":"object"}`,
		},
		{
			name: "an empty required entry is pruned and the declared one kept",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"required":["a",""]}`,
			want: `{"properties":{"a":{"type":"string"}},"required":["a"],"type":"object"}`,
		},
		{
			name: "required with nothing left to keep goes away",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"required":["b"]}`,
			want: `{"properties":{"a":{"type":"string"}},"type":"object"}`,
		},
		{
			name: "required without properties asserts nothing the enforcer can use",
			in:   `{"type":"object","required":["a"]}`,
			want: `{"type":"object"}`,
		},
		{
			name: "required on a non-object is a no-op and is dropped",
			in:   `{"type":"string","required":["a"]}`,
			want: `{"type":"string"}`,
		},
		{
			name: "enum values the type rules out are dropped, the rest kept",
			in:   `{"type":"object","properties":{"v":{"type":"string","enum":["a",1,"b"]}}}`,
			want: `{"properties":{"v":{"enum":["a","b"],"type":"string"}},"type":"object"}`,
		},
		{
			// Nothing satisfies the pair, and one of them has to give. The type is
			// kept because it is the tighter of the two survivors: keeping the enum
			// instead would accept a boolean the schema ruled out.
			name: "an enum the type rules out entirely gives way to the type",
			in:   `{"type":"object","properties":{"v":{"type":["null"],"enum":[false]}}}`,
			want: `{"properties":{"v":{"type":["null"]}},"type":"object"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, level := range []ValidateLevel{ValidateLevelLoose, ValidateLevelLite, ValidateLevelStrict} {
				if err := newSchemaValidator(WithValidateLevel(level)).Validate(tc.in); err != nil {
					t.Errorf("%s must accept the schema, got %v", level, err)
				}
			}

			parsed, err := ParseSchema(tc.in)
			if err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}
			got, _ := parsed.Canonical()
			if got != tc.want {
				t.Errorf("canonical form\n got %s\nwant %s", got, tc.want)
			}
		})
	}
}

// A path naming an anyOf branch has to be followed into that branch even when it
// is the only part of the path. Treating it as a keyword on the root instead made
// every simplification of a root-level branch fail and throw away the whole
// document.
func TestSimplifyReachesARootLevelAnyOfBranch(t *testing.T) {
	const schema = `{"anyOf":[{"$ref":"#/$defs/S","type":"string"}],"$defs":{"S":{"type":"string","maxLength":3}}}`

	parsed, err := ParseSchema(schema)
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	// The branch restates the definition's own type, so dropping it costs nothing.
	// What matters is that the rest of the document is still there: before the fix
	// the failed lookup returned an empty schema.
	got, _ := parsed.Canonical()
	const want = `{"$defs":{"S":{"maxLength":3,"type":"string"}},"anyOf":[{"$ref":"#/$defs/S"}]}`
	if got != want {
		t.Errorf("canonical form\n got %s\nwant %s", got, want)
	}
}

// The empty string is a property name like any other: 2020-12 puts no constraint
// on the keys of properties, and the enforcer generates it, down to {"": ...}.
// walle used to delete such a property while canonicalising, silently dropping a
// field the caller had declared.
func TestEmptyPropertyNameSurvives(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "declared as a property",
			in:   `{"type":"object","properties":{"":{"type":"string"},"a":{"type":"number"}}}`,
			want: `{"properties":{"":{"type":"string"},"a":{"type":"number"}},"type":"object"}`,
		},
		{
			name: "declared and required",
			in:   `{"type":"object","properties":{"":{"type":"string"}},"required":[""],"additionalProperties":false}`,
			want: `{"additionalProperties":false,"properties":{"":{"type":"string"}},"required":[""],"type":"object"}`,
		},
		{
			// Undeclared is the one thing that is still wrong, and it is wrong for
			// the same reason any other undeclared name is: only that entry goes.
			name: "required but never declared",
			in:   `{"type":"object","properties":{"a":{"type":"number"}},"required":["","a"]}`,
			want: `{"properties":{"a":{"type":"number"}},"required":["a"],"type":"object"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, level := range []ValidateLevel{ValidateLevelLite, ValidateLevelStrict} {
				if err := newSchemaValidator(WithValidateLevel(level)).Validate(tc.in); err != nil {
					t.Errorf("%s must accept the schema, got %v", level, err)
				}
			}

			parsed, err := ParseSchema(tc.in)
			if err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}
			got, _ := parsed.Canonical()
			if got != tc.want {
				t.Errorf("canonical form\n got %s\nwant %s", got, tc.want)
			}
		})
	}
}

// A path whose only part names an anyOf branch, such as "anyOf{0}", has to be
// followed into that branch. Resolving it as a keyword on the root instead made
// every simplification inside a root-level branch fail and empty the whole
// document, which is the opposite of degrading just the field at fault.
func TestSimplifyDegradesInsideARootLevelAnyOfBranch(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "an undeclared required entry inside a branch",
			in:   `{"anyOf":[{"type":"object","properties":{"a":{"type":"string"}},"required":["a","b"]}]}`,
			want: `{"anyOf":[{"properties":{"a":{"type":"string"}},"required":["a"],"type":"object"}]}`,
		},
		{
			name: "required inside a branch with nothing left to keep",
			in:   `{"anyOf":[{"type":"object","properties":{"a":{"type":"string"}},"required":["b"]}]}`,
			want: `{"anyOf":[{"properties":{"a":{"type":"string"}},"type":"object"}]}`,
		},
		{
			name: "an enum value the branch's type rules out",
			in:   `{"anyOf":[{"type":"string","enum":["a",1]}]}`,
			want: `{"anyOf":[{"enum":["a"],"type":"string"}]}`,
		},
		{
			name: "an enum the branch's type rules out entirely",
			in:   `{"anyOf":[{"type":["null"],"enum":[false]}]}`,
			want: `{"anyOf":[{"type":["null"]}]}`,
		},
		{
			name: "a nested anyOf carrying its own parent constraint",
			in:   `{"anyOf":[{"type":"string","anyOf":[{"minLength":1},{"maxLength":9}]}]}`,
			want: `{"anyOf":[{"anyOf":[{"minLength":1,"type":"string"},{"maxLength":9,"type":"string"}]}]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseSchema(tc.in)
			if err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}
			got, _ := parsed.Canonical()
			if got != tc.want {
				t.Errorf("canonical form\n got %s\nwant %s", got, tc.want)
			}
		})
	}
}

// A chain of definitions where every level references the next through two
// required properties forms a DAG, not a cycle: every definition terminates,
// so the schema is valid. Walking each sibling's subgraph from scratch costs
// 2^n walks -- n=30 would take hours; reusing expanded definitions brings it
// back to milliseconds.
func TestRefTerminationReusesExpandedDefs(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"type":"object","properties":{"root":{"$ref":"#/$defs/D1"}},"required":["root"],"$defs":{`)
	const levels = 30
	for i := 1; i < levels; i++ {
		if i > 1 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `"D%d":{"type":"object","properties":{"p":{"$ref":"#/$defs/D%d"},"q":{"$ref":"#/$defs/D%d"}},"required":["p","q"]}`, i, i+1, i+1)
	}
	fmt.Fprintf(&sb, `,"D%d":{"type":"string"}}}`, levels)

	schema, err := ParseSchema(sb.String())
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	if err := schema.Validate(); err != nil {
		t.Fatalf("chain of terminating definitions should validate: %v", err)
	}
}

// Caching a termination verdict is only sound when computing it never ran into
// the entry stack. Here D2's verdict computed while entering through D1 is cut
// short by the stack (D2 -> D1 -> D2), but standalone D2 terminates because
// D1's second anyOf branch is a plain string. A poisoned cache would reject
// this schema; it is valid: {"p":"s","q":{"y":"s"}} satisfies it.
func TestRefTerminationVerdictIsNotCachedAcrossEntryStacks(t *testing.T) {
	raw := `{
		"type": "object",
		"properties": {
			"p": {"$ref": "#/$defs/D1"},
			"q": {"$ref": "#/$defs/D2"}
		},
		"required": ["p", "q"],
		"$defs": {
			"D1": {
				"anyOf": [
					{"type": "object", "properties": {"x": {"$ref": "#/$defs/D2"}}, "required": ["x"]},
					{"type": "string"}
				]
			},
			"D2": {"type": "object", "properties": {"y": {"$ref": "#/$defs/D1"}}, "required": ["y"]}
		}
	}`

	schema, err := ParseSchema(raw)
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	if err := schema.Validate(); err != nil {
		t.Fatalf("schema with a finite instance should validate: %v", err)
	}
}

// A diamond chain that closes a cycle has no finite instance, but proving it by
// walking costs 2^n walks: every path is cut by the cycle guard, so nothing is
// memoizable. The shared step budget fails closed instead of hanging.
func TestRefTerminationFailsClosedBeyondStepBudget(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"type":"object","properties":{"root":{"$ref":"#/$defs/D1"}},"required":["root"],"$defs":{`)
	const levels = 25
	for i := 1; i < levels; i++ {
		if i > 1 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `"D%d":{"type":"object","properties":{"p":{"$ref":"#/$defs/D%d"},"q":{"$ref":"#/$defs/D%d"}},"required":["p","q"]}`, i, i+1, i+1)
	}
	fmt.Fprintf(&sb, `,"D%d":{"type":"object","properties":{"back":{"$ref":"#/$defs/D1"}},"required":["back"]}}}`, levels)

	schema, err := ParseSchema(sb.String())
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	err = schema.Validate()
	if err == nil {
		t.Fatal("expected rejection of a cycle with no finite instance")
	}
	if !strings.Contains(err.Error(), "too complex") && !strings.Contains(err.Error(), "infinite recursion") {
		t.Fatalf("unexpected error: %v", err)
	}
}
