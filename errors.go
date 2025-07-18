package walle

import (
	"fmt"
	"regexp"
)

type SchemaError struct {
	Path         string       `json:"path,omitempty"`
	Message      string       `json:"message,omitempty"`
	SimplifyFunc SimplifyFunc `json:"-"`
}

func (e *SchemaError) Error() string {
	// Remove index from anyOf paths
	path := regexp.MustCompile(`anyOf\{[^}]+\}`).ReplaceAllString(e.Path, "anyOf")
	return fmt.Sprintf("At path '%s': %s", path, e.Message)
}

func NewSchemaError(message string, path string, simplifyFunc SimplifyFunc) error {
	return &SchemaError{
		Path:         path,
		Message:      message,
		SimplifyFunc: simplifyFunc,
	}
}

type UnmarshalError struct {
	Err error `json:"error,omitempty"`
}

func (e *UnmarshalError) Error() string {
	return fmt.Sprintf("JSON schema parsing error: %s", e.Err.Error())
}

func NewUnmarshalError(err error) *UnmarshalError {
	return &UnmarshalError{
		Err: err,
	}
}

func IsSchemaError(err error) bool {
	_, ok := err.(*SchemaError)
	return ok
}

func IsUnmarshalError(err error) bool {
	_, ok := err.(*UnmarshalError)
	return ok
}
