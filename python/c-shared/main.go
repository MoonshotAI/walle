package main

// #include <stdlib.h>
import "C"

import (
	"encoding/json"
	"fmt"

	"unsafe"

	"github.com/moonshotai/walle"
)

//export ValidateSchema
func ValidateSchema(schemaStr *C.char, configStr *C.char) *C.char {
	var errMsg string
	goSchemaStr := C.GoString(schemaStr)
	goConfigStr := C.GoString(configStr)

	schema, err := walle.ParseSchema(goSchemaStr)
	if err != nil {
		errMsg = err.Error()
		return C.CString(errMsg)
	}

	config := walle.DefaultValidatorConfig()
	if goConfigStr != "" {
		if err := json.Unmarshal([]byte(goConfigStr), &config); err != nil {
			errMsg = fmt.Sprintf("invalid config format: %v", err)
			return C.CString(errMsg)
		}
	}

	option := func(c *walle.SchemaValidatorConfig) {
		*c = config
	}

	if err = schema.Validate(option); err != nil {
		errMsg = err.Error()
	}

	return C.CString(errMsg)
}

//export CanonicalSchema
func CanonicalSchema(schemaStr *C.char) *C.char {
	goSchemaStr := C.GoString(schemaStr)
	type payload struct {
		Error     string `json:"error,omitempty"`
		Canonical string `json:"canonical,omitempty"`
		Warning   string `json:"warning,omitempty"`
	}
	out := payload{}
	schema, err := walle.ParseSchema(goSchemaStr)
	if err != nil {
		out.Error = err.Error()
		b, _ := json.Marshal(out)
		return C.CString(string(b))
	}
	canonicalStr, warnErr := schema.Canonical()
	out.Canonical = canonicalStr
	if warnErr != nil {
		out.Warning = warnErr.Error()
	}
	b, err := json.Marshal(out)
	if err != nil {
		out = payload{Error: fmt.Sprintf("marshal canonical response: %v", err)}
		b, _ = json.Marshal(out)
	}
	return C.CString(string(b))
}

//export FreeErrString
func FreeErrString(str *C.char) {
	if str != nil {
		C.free(unsafe.Pointer(str))
	}
}

func main() {}
