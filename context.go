package walle

// validationContext manages state during validation
type validationContext struct {
	SchemaRoot map[string]any      `json:"schema_root,omitempty"`
	RefPaths   map[string]struct{} `json:"ref_paths,omitempty"`
}

func newValidationContext() *validationContext {
	return &validationContext{
		RefPaths: make(map[string]struct{}),
	}
}

func (c *validationContext) RaiseError(message string, path schemaPath) error {
	return NewSchemaError(message, path.String(), nil)
}

func (c *validationContext) RaiseErrorWithSimplify(message string, path schemaPath, simplifyFunc SimplifyFunc) error {
	return NewSchemaError(message, path.String(), simplifyFunc)
}
