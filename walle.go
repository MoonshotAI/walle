package walle

type Schema map[string]any

func ParseSchema(jsonStr string) (Schema, error) {
	return parseSchema(jsonStr)
}

func (s Schema) Validate(options ...SchemaValidatorOption) error {
	validator := newSchemaValidator(options...)
	if validator.config.IsLoose() {
		return nil
	}
	return validator.Validate(s)
}

// Canonical returns a schema representation that conforms to Moonshot AI server requirements.
// It uses `strict` validation level, which is the most permissive level supported by the enforcer-server.
// If the original schema has issues, it returns a simplified schema.
func (s Schema) Canonical() (string, error) {
	validator := newSchemaValidator(WithValidateLevel(ValidateLevelUltra))
	return validator.CanonicalWithMaxAttempts(s, 20)
}
