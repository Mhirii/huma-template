package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// BindConfigStruct configures a Viper instance to bind configuration from multiple sources
// (flags, environment variables, and YAML) to a struct using struct tags.
//
// The function introspects the provided struct and registers configuration sources based on
// the following struct field tags:
//   - "flag": Name of the command-line flag (e.g., `flag:"port"`)
//   - "env": Name of the environment variable (e.g., `env:"APP_PORT"`)
//   - "yaml": Key in YAML configuration files (e.g., `yaml:"port"`)
//   - "default": Default value if no other source provides a value (e.g., `default:"8080"`)
//
// Parameters:
//   - v: A pointer to a viper.Viper instance that will be configured
//   - s: A struct (or pointer to a struct) whose fields define the configuration schema
//   - prefix: A string prefix to prepend to YAML keys for nested configuration structures
//     (e.g., "server" would make a field with yaml tag "port" resolve to "server.port")
//
// Behavior:
//   - If a field's type is not supported for flag registration (only int, int64, string, bool,
//     and float64 are supported), an error message is written to stderr
//   - Default values are registered with Viper and will be used if no flag, environment
//     variable, or configuration file provides a value
//   - Environment variables are bound to configuration keys with the highest precedence
//   - Flags are registered with pflag and can override environment variables and defaults
//
// Example struct:
//
//	type AppConfig struct {
//	    Port int    `flag:"port" env:"APP_PORT" yaml:"port" default:"8080"`
//	    Host string `flag:"host" env:"APP_HOST" yaml:"host" default:"localhost"`
//	}
//
// Load Order (lowest to highest priority):
// 1. Default values (from "default" tag) - lowest priority
// 2. YAML configuration files (from "yaml" tag)
// 3. Environment variables (from "env" tag)
// 4. Command-line flags (from "flag" tag) - highest priority
func BindConfigStruct(v *viper.Viper, s interface{}, prefix string) {
	l := logging.L()
	t := reflect.TypeOf(s)
	vStruct := reflect.ValueOf(s)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		vStruct = vStruct.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fieldName := f.Name
		flagTag := f.Tag.Get("flag")
		envTag := f.Tag.Get("env")
		yamlTag := f.Tag.Get("yaml")
		defaultTag := f.Tag.Get("default")

		// Compose key with prefix for nested structs
		key := yamlTag
		if prefix != "" {
			key = prefix + "." + yamlTag
		}

		// Register flag (highest priority in load order)
		if flagTag != "" {
			l.Debug().Msgf("Registering flag for field '%s': flag=%s, env=%s, yaml=%s, default=%s", fieldName, flagTag, envTag, yamlTag, defaultTag)

			switch f.Type.Kind() {
			case reflect.Int, reflect.Int64:
				defVal := 0
				if defaultTag != "" {
					if intVal, err := strconv.Atoi(defaultTag); err == nil {
						defVal = intVal
					}
				}
				pflag.Int(flagTag, defVal, fmt.Sprintf("%s (default: %s)", fieldName, defaultTag))
			case reflect.String:
				defVal := ""
				if defaultTag != "" {
					defVal = defaultTag
				}
				pflag.String(flagTag, defVal, fmt.Sprintf("%s (default: %s)", fieldName, defaultTag))
			case reflect.Bool:
				defVal := false
				if defaultTag != "" {
					defVal = defaultTag == "true"
				}
				pflag.Bool(flagTag, defVal, fmt.Sprintf("%s (default: %s)", fieldName, defaultTag))
			case reflect.Float64:
				defVal := 0.0
				if defaultTag != "" {
					if floatVal, err := strconv.ParseFloat(defaultTag, 64); err == nil {
						defVal = floatVal
					}
				}
				pflag.Float64(flagTag, defVal, fmt.Sprintf("%s (default: %s)", fieldName, defaultTag))
			default:
				fmt.Fprintf(os.Stderr, "Unsupported flag type for %s: %s\n", fieldName, f.Type.Kind())
			}
		}
		// Set default (lowest priority in load order)
		if defaultTag != "" {
			l.Debug().Msgf("Setting default for key '%s': %s", key, defaultTag)
			v.SetDefault(key, defaultTag)
		}
		// Bind env (second highest priority in load order)
		if envTag != "" {
			v.BindEnv(key, envTag)
		}
	}
}

// ValidateConfigStruct validates a struct's fields against validation rules defined in struct tags.
//
// The function introspects the provided struct and validates each field based on rules specified
// in the "validate" struct tag. It supports recursive validation of nested structs.
//
// Validation Rules:
// The "validate" tag can contain one or more comma-separated rules:
//   - "required": Field must not be empty/zero. Skipped if a "default" tag is present.
//   - "min=<value>": For numeric fields, the value must be >= the specified minimum.
//   - "max=<value>": For numeric fields, the value must be <= the specified maximum.
//   - "oneof=<opt1> <opt2> ...": For string fields, the value must match one of the space-separated options.
//
// Parameters:
//   - s: A struct (or pointer to a struct) to validate
//
// Return:
//   - nil if all validation rules pass
//   - An error describing the first validation failure, including the field name and reason
//
// Behavior:
//   - Fields without a "validate" tag are skipped
//   - Nested structs are recursively validated with error messages prefixed by parent field name
//   - The "required" rule respects default values; if a default is set, the field is not required
//   - For "min=" and "max=" rules, values are converted to integers using the Atoi helper
//   - For "oneof=" rules, string values are matched exactly against the provided options
//
// Example struct:
//
//	type AppConfig struct {
//	    Port int    `validate:"required,min=1,max=65535"`
//	    Mode string `validate:"required,oneof=debug release"`
//	    Timeout int `validate:"min=0,max=300" default:"30"`
//	}
func ValidateConfigStruct(s interface{}) error {
	t := reflect.TypeOf(s)
	vStruct := reflect.ValueOf(s)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		vStruct = vStruct.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		val := vStruct.Field(i)
		// Recursively validate nested structs
		if f.Type.Kind() == reflect.Struct {
			if err := ValidateConfigStruct(val.Interface()); err != nil {
				return fmt.Errorf("%s: %w", f.Name, err)
			}
			continue
		}
		validateTag := f.Tag.Get("validate")
		if validateTag == "" {
			continue
		}
		for _, rule := range strings.Split(validateTag, ",") {
			rule = strings.TrimSpace(rule)
			if rule == "required" && IsZero(val) {
				if f.Tag.Get("default") == "" {
					return fmt.Errorf("%s is required", f.Name)
				}
			}
			if after, ok := strings.CutPrefix(rule, "min="); ok {
				min := Atoi(after)
				if val.Int() < int64(min) {
					return fmt.Errorf("%s must be >= %d", f.Name, min)
				}
			}
			if after, ok := strings.CutPrefix(rule, "max="); ok {
				max := Atoi(after)
				if val.Int() > int64(max) {
					return fmt.Errorf("%s must be <= %d", f.Name, max)
				}
			}
			if after, ok := strings.CutPrefix(rule, "oneof="); ok {
				opts := strings.Split(after, " ")
				found := false
				for _, opt := range opts {
					if val.String() == opt {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("%s must be one of %v", f.Name, opts)
				}
			}
		}
	}
	return nil
}

func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int64:
		return v.Int() == 0
	}
	return false
}

func Atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
