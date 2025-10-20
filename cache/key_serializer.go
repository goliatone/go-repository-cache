package cache

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// KeySeparator defines the delimiter used between cache key segments.
const KeySeparator = "::"

// defaultKeySerializer implements KeySerializer using reflection-based serialization.
// It handles function pointers using %p formatting, recursive slices, and falls back to JSON
// for complex types while ensuring deterministic key generation across runs.
type defaultKeySerializer struct{}

// NewDefaultKeySerializer creates a new instance of the default key serializer.
func NewDefaultKeySerializer() KeySerializer {
	return &defaultKeySerializer{}
}

// SerializeKey builds a cache key from method name and args using reflection.
// It produces stable keys across runs by handling various Go types deterministically.
func (s *defaultKeySerializer) SerializeKey(method string, args ...any) string {
	if len(args) == 0 {
		return method
	}

	var parts []string
	parts = append(parts, method)

	for _, arg := range args {
		serialized := s.serializeValue(arg)
		parts = append(parts, serialized)
	}

	return strings.Join(parts, KeySeparator)
}

// serializeValue handles individual argument serialization based on type.
func (s *defaultKeySerializer) serializeValue(v any) string {
	if v == nil {
		return "nil"
	}

	rv := reflect.ValueOf(v)
	rt := reflect.TypeOf(v)

	// Handle function pointers using %p formatting for stability
	if rt.Kind() == reflect.Func {
		return fmt.Sprintf("func:%p", v)
	}

	// Handle pointers by dereferencing
	if rt.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return "nil"
		}
		return s.serializeValue(rv.Elem().Interface())
	}

	// Handle slices recursively
	if rt.Kind() == reflect.Slice {
		if rv.IsNil() {
			return "slice:nil"
		}
		return s.serializeSlice(rv)
	}

	// Handle arrays
	if rt.Kind() == reflect.Array {
		return s.serializeArray(rv)
	}

	// Handle maps with sorted keys for determinism
	if rt.Kind() == reflect.Map {
		if rv.IsNil() {
			return "map:nil"
		}
		return s.serializeMap(rv)
	}

	// Handle structs
	if rt.Kind() == reflect.Struct {
		return s.serializeStruct(rv, rt)
	}

	// Handle channels, interfaces with special formatting
	switch rt.Kind() {
	case reflect.Chan:
		return fmt.Sprintf("chan:%p", v)
	case reflect.Interface:
		if rv.IsNil() {
			return "interface:nil"
		}
		return s.serializeValue(rv.Elem().Interface())
	}

	// For basic types, use string representation
	if s.isBasicType(rt.Kind()) {
		return fmt.Sprintf("%v", v)
	}

	// Fallback to JSON for complex types
	return s.jsonFallback(v)
}

// serializeSlice handles slice serialization recursively
func (s *defaultKeySerializer) serializeSlice(rv reflect.Value) string {
	length := rv.Len()
	parts := make([]string, length)

	for i := 0; i < length; i++ {
		elem := rv.Index(i).Interface()
		parts[i] = s.serializeValue(elem)
	}

	return fmt.Sprintf("slice[%d]:{%s}", length, strings.Join(parts, ","))
}

// serializeArray handles array serialization
func (s *defaultKeySerializer) serializeArray(rv reflect.Value) string {
	length := rv.Len()
	parts := make([]string, length)

	for i := 0; i < length; i++ {
		elem := rv.Index(i).Interface()
		parts[i] = s.serializeValue(elem)
	}

	return fmt.Sprintf("array[%d]:{%s}", length, strings.Join(parts, ","))
}

// serializeMap handles map serialization with sorted keys for determinism
func (s *defaultKeySerializer) serializeMap(rv reflect.Value) string {
	keys := rv.MapKeys()

	// Sort keys for deterministic output
	keyStrings := make([]string, len(keys))
	for i, k := range keys {
		keyStrings[i] = s.serializeValue(k.Interface())
	}
	sort.Strings(keyStrings)

	// Build key-value pairs in sorted order
	pairs := make([]string, len(keyStrings))
	for i, keyStr := range keyStrings {
		// Find the original key that matches this string representation
		var originalKey reflect.Value
		for _, k := range keys {
			if s.serializeValue(k.Interface()) == keyStr {
				originalKey = k
				break
			}
		}

		value := rv.MapIndex(originalKey)
		valueStr := s.serializeValue(value.Interface())
		pairs[i] = fmt.Sprintf("%s=%s", keyStr, valueStr)
	}

	return fmt.Sprintf("map[%d]:{%s}", len(pairs), strings.Join(pairs, ","))
}

// serializeStruct handles struct serialization with field names
func (s *defaultKeySerializer) serializeStruct(rv reflect.Value, rt reflect.Type) string {
	numFields := rv.NumField()
	parts := make([]string, 0, numFields)

	for i := 0; i < numFields; i++ {
		field := rt.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldValue := rv.Field(i)
		if !fieldValue.CanInterface() {
			continue
		}

		serializedValue := s.serializeValue(fieldValue.Interface())
		parts = append(parts, fmt.Sprintf("%s:%s", field.Name, serializedValue))
	}

	return fmt.Sprintf("struct:{%s}", strings.Join(parts, ","))
}

// isBasicType checks if a kind represents a basic Go type
func (s *defaultKeySerializer) isBasicType(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true
	default:
		return false
	}
}

// jsonFallback provides JSON serialization as a last resort
func (s *defaultKeySerializer) jsonFallback(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		// If JSON marshaling fails, use type and pointer info
		rv := reflect.ValueOf(v)
		rt := reflect.TypeOf(v)
		if rv.CanAddr() {
			return fmt.Sprintf("fallback:%s:%x", rt.String(), rv.UnsafeAddr())
		}
		return fmt.Sprintf("fallback:%s", rt.String())
	}
	return fmt.Sprintf("json:%s", string(data))
}
