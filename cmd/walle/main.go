package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/mattn/go-isatty"
	"github.com/moonshotai/walle"
)

var (
	schema     = flag.String("schema", "", "JSON Schema string")
	schemaFile = flag.String("schema-file", "", "JSON Schema file path")
	level      = flag.String("level", "strict", "Validation level (loose, lite, strict)")
	h          = flag.Bool("h", false, "Show help information")
	v          = flag.Bool("v", false, "Show version information")
)

func main() {
	flag.Parse()

	if *h {
		printUsage()
		return
	}

	if *v {
		printVersion()
		return
	}

	var schemaContent string
	var err error

	if *schema != "" && *schemaFile != "" {
		fmt.Fprintln(os.Stderr, "Error: cannot use both -schema and -schema-file")
		printUsage()
		os.Exit(1)
	}

	if *schema != "" {
		schemaContent = *schema
	} else if *schemaFile != "" {
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
	} else {
		// Check if stdin is a terminal (interactive mode)
		if isatty.IsTerminal(os.Stdin.Fd()) {
			printUsage()
			os.Exit(1)
		}

		// Read from stdin (pipe or redirect)
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: unable to read from stdin: %v\n", err)
			os.Exit(1)
		}
		schemaContent = string(content)

		if schemaContent == "" {
			fmt.Fprintln(os.Stderr, "Error: no schema content received from stdin")
			os.Exit(1)
		}
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
	fmt.Println("  walle [-level <validation-level>] < schema.json")
	fmt.Println("  cat schema.json | walle [-level <validation-level>]")
	fmt.Println()
	fmt.Println("Parameters:")
	fmt.Println("  -schema        JSON Schema string")
	fmt.Println("  -schema-file   JSON Schema file path")
	fmt.Println("  -level         Validation level (loose, lite, strict), default is strict")
	fmt.Println("  -h             Show this help information")
	fmt.Println("  -v             Show version information")
	fmt.Println()
	fmt.Println("If neither -schema nor -schema-file is provided, schema will be read from stdin.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  walle -schema '{\"type\": \"object\"}' -level strict")
	fmt.Println("  walle -schema-file schema.json")
	fmt.Println("  walle < schema.json")
	fmt.Println("  cat schema.json | walle -level strict")
	fmt.Println("  echo '{\"type\": \"string\"}' | walle")
	fmt.Println("  walle -v")
}

func printVersion() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("walle version unknown")
		return
	}

	// auto get version info
	version := info.Main.Version
	if version == "" || version == "(devel)" {
		version = "dev"
	}

	fmt.Printf("walle version %s\n", version)
	fmt.Printf("module: %s\n", info.Main.Path)

	// show commit info (if any)
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			if len(setting.Value) > 7 {
				fmt.Printf("commit: %s\n", setting.Value[:7])
			} else {
				fmt.Printf("commit: %s\n", setting.Value)
			}
			break
		}
	}
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
