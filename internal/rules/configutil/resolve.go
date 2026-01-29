// Package configutil provides utilities for rule configuration resolution.
package configutil

import (
	"reflect"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

// Resolve merges user options over defaults and unmarshals to typed config.
// If opts is nil or empty, returns defaults unchanged.
// This eliminates duplicated map-to-struct conversion in each rule.
func Resolve[T any](opts map[string]any, defaults T) T {
	if len(opts) == 0 {
		return defaults
	}

	k := koanf.New(".")
	if err := k.Load(confmap.Provider(opts, "."), nil); err != nil {
		return defaults
	}

	var result T
	if err := k.Unmarshal("", &result); err != nil {
		return defaults
	}

	// Merge defaults for zero-valued fields
	return mergeDefaults(result, defaults)
}

// mergeDefaults fills zero-valued fields in result with values from defaults.
func mergeDefaults[T any](result, defaults T) T {
	resultVal := reflect.ValueOf(&result).Elem()
	defaultsVal := reflect.ValueOf(defaults)

	if resultVal.Kind() != reflect.Struct {
		return result
	}

	for i := range resultVal.NumField() {
		field := resultVal.Field(i)
		if !field.CanSet() {
			continue
		}
		if isZero(field) {
			field.Set(defaultsVal.Field(i))
		}
	}

	return result
}

// isZero checks if a reflect.Value is the zero value for its type.
func isZero(v reflect.Value) bool {
	//exhaustive:ignore
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Map:
		return v.IsNil()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}
