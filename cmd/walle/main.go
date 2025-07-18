package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/moonshotai/walle"
)

var (
	schema     = flag.String("schema", "", "JSON Schema string")
	schemaFile = flag.String("schema-file", "", "JSON Schema file path")
	level      = flag.String("level", "strict", "Validation level (loose, lite, strict)")
	help       = flag.Bool("help", false, "Show help information")
)

func main() {
	flag.Parse()

	if *help {
		printUsage()
		return
	}

	var schemaContent string
	var err error

	if *schema != "" && *schemaFile != "" {
		fmt.Fprintln(os.Stderr, "Error: cannot use both -schema and -schema-file")
		printUsage()
		os.Exit(1)
	}

	if *schema == "" && *schemaFile == "" {
		fmt.Fprintln(os.Stderr, "Error: must provide either -schema or -schema-file")
		printUsage()
		os.Exit(1)
	}

	if *schema != "" {
		schemaContent = *schema
	}

	if *schemaFile != "" {
		file, err := os.Open(*schemaFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: unable to open file %s: %v\n", *schemaFile, err)
			os.Exit(1)
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: unable to read file %s: %v\n", *schemaFile, err)
			os.Exit(1)
		}
		schemaContent = string(content)
	}

	if !isValidLevel(*level) {
		fmt.Fprintf(os.Stderr, "Error: invalid validation level '%s'. Valid values: loose, lite, strict\n", *level)
		os.Exit(1)
	}

	walleSchema, err := walle.ParseSchema(schemaContent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to parse schema: %v\n", err)
		os.Exit(1)
	}

	validateLevelEnum := walle.ValidateLevel(*level)
	option := walle.WithValidateLevel(validateLevelEnum)

	if err := walleSchema.Validate(option); err != nil {
		fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Validation successful!")
}

func printUsage() {
	fmt.Println("walle - Moonshot AI flavored Json schema validator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  walle -schema <schema-string> [-level <validation-level>]")
	fmt.Println("  walle -schema-file <file-path> [-level <validation-level>]")
	fmt.Println()
	fmt.Println("Parameters:")
	fmt.Println("  -schema        JSON Schema string")
	fmt.Println("  -schema-file   JSON Schema file path")
	fmt.Println("  -level Validation level (loose, lite, strict), default is strict")
	fmt.Println("  -help          Show this help information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  walle -schema '{\"type\": \"object\"}' -level strict")
	fmt.Println("  walle -schema-file schema.json")
}

func isValidLevel(level string) bool {
	validLevels := []string{"loose", "lite", "strict"}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}
