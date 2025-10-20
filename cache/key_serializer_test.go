package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScenario represents a test scenario loaded from fixtures
type TestScenario struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Cases       []TestCase `json:"cases"`
}

// TestCase represents individual test cases within a scenario
type TestCase struct {
	Method      string        `json:"method"`
	Args        []interface{} `json:"args"`
	ExpectedKey string        `json:"expectedKey"`
}

// TestFixtures represents the structure of the test fixture file
type TestFixtures struct {
	Scenarios []TestScenario `json:"scenarios"`
}

func joinWithSeparator(parts ...string) string {
	return strings.Join(parts, KeySeparator)
}

func TestDefaultKeySerializer_BasicTypes(t *testing.T) {
	serializer := NewDefaultKeySerializer()

	tests := []struct {
		name   string
		method string
		args   []any
		want   string
	}{
		{
			name:   "no args",
			method: "List",
			args:   []any{},
			want:   "List",
		},
		{
			name:   "single int",
			method: "GetByID",
			args:   []any{42},
			want:   joinWithSeparator("GetByID", "42"),
		},
		{
			name:   "multiple basic types",
			method: "Get",
			args:   []any{1, "hello", true, 3.14},
			want:   joinWithSeparator("Get", "1", "hello", "true", "3.14"),
		},
		{
			name:   "string with special chars",
			method: "Search",
			args:   []any{"hello:world"},
			want:   joinWithSeparator("Search", "hello:world"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializer.SerializeKey(tt.method, tt.args...)
			if got != tt.want {
				t.Errorf("SerializeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultKeySerializer_NilValues(t *testing.T) {
	serializer := NewDefaultKeySerializer()

	tests := []struct {
		name   string
		method string
		args   []any
		want   string
	}{
		{
			name:   "nil interface",
			method: "GetByPtr",
			args:   []any{nil},
			want:   joinWithSeparator("GetByPtr", "nil"),
		},
		{
			name:   "nil pointer",
			method: "GetByRef",
			args:   []any{(*int)(nil)},
			want:   joinWithSeparator("GetByRef", "nil"),
		},
		{
			name:   "nil slice",
			method: "GetBySlice",
			args:   []any{([]int)(nil)},
			want:   joinWithSeparator("GetBySlice", "slice:nil"),
		},
		{
			name:   "nil map",
			method: "GetByMap",
			args:   []any{(map[string]int)(nil)},
			want:   joinWithSeparator("GetByMap", "map:nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializer.SerializeKey(tt.method, tt.args...)
			if got != tt.want {
				t.Errorf("SerializeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultKeySerializer_Slices(t *testing.T) {
	serializer := NewDefaultKeySerializer()

	tests := []struct {
		name   string
		method string
		args   []any
		want   string
	}{
		{
			name:   "empty slice",
			method: "GetByIDs",
			args:   []any{[]int{}},
			want:   joinWithSeparator("GetByIDs", "slice[0]:{}"),
		},
		{
			name:   "int slice",
			method: "GetByIDs",
			args:   []any{[]int{1, 2, 3}},
			want:   joinWithSeparator("GetByIDs", "slice[3]:{1,2,3}"),
		},
		{
			name:   "string slice",
			method: "GetByNames",
			args:   []any{[]string{"alice", "bob"}},
			want:   joinWithSeparator("GetByNames", "slice[2]:{alice,bob}"),
		},
		{
			name:   "nested slice",
			method: "GetByMatrix",
			args:   []any{[][]int{{1, 2}, {3, 4}}},
			want:   joinWithSeparator("GetByMatrix", "slice[2]:{slice[2]:{1,2},slice[2]:{3,4}}"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializer.SerializeKey(tt.method, tt.args...)
			if got != tt.want {
				t.Errorf("SerializeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultKeySerializer_Arrays(t *testing.T) {
	serializer := NewDefaultKeySerializer()

	tests := []struct {
		name   string
		method string
		args   []any
		want   string
	}{
		{
			name:   "int array",
			method: "GetByArray",
			args:   []any{[3]int{1, 2, 3}},
			want:   joinWithSeparator("GetByArray", "array[3]:{1,2,3}"),
		},
		{
			name:   "string array",
			method: "GetByStrArray",
			args:   []any{[2]string{"hello", "world"}},
			want:   joinWithSeparator("GetByStrArray", "array[2]:{hello,world}"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializer.SerializeKey(tt.method, tt.args...)
			if got != tt.want {
				t.Errorf("SerializeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultKeySerializer_Maps(t *testing.T) {
	serializer := NewDefaultKeySerializer()

	tests := []struct {
		name   string
		method string
		args   []any
		want   string
	}{
		{
			name:   "empty map",
			method: "GetByFilters",
			args:   []any{map[string]int{}},
			want:   joinWithSeparator("GetByFilters", "map[0]:{}"),
		},
		{
			name:   "string to int map",
			method: "GetByFilters",
			args:   []any{map[string]int{"age": 25, "count": 10}},
			want:   joinWithSeparator("GetByFilters", "map[2]:{age=25,count=10}"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializer.SerializeKey(tt.method, tt.args...)
			if got != tt.want {
				t.Errorf("SerializeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultKeySerializer_Structs(t *testing.T) {
	serializer := NewDefaultKeySerializer()

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	type UserWithPrivate struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		password string // unexported field should be ignored
	}

	tests := []struct {
		name   string
		method string
		args   []any
		want   string
	}{
		{
			name:   "simple struct",
			method: "GetUser",
			args:   []any{User{ID: 1, Name: "alice"}},
			want:   joinWithSeparator("GetUser", "struct:{ID:1,Name:alice}"),
		},
		{
			name:   "struct with private field",
			method: "GetUserPrivate",
			args:   []any{UserWithPrivate{ID: 2, Name: "bob", password: "secret"}},
			want:   joinWithSeparator("GetUserPrivate", "struct:{ID:2,Name:bob}"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializer.SerializeKey(tt.method, tt.args...)
			if got != tt.want {
				t.Errorf("SerializeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultKeySerializer_Functions(t *testing.T) {
	serializer := NewDefaultKeySerializer()

	testFunc := func() {}

	// Test that function pointers produce deterministic keys with %p formatting
	key1 := serializer.SerializeKey("GetWithFunc", testFunc)
	key2 := serializer.SerializeKey("GetWithFunc", testFunc)

	if key1 != key2 {
		t.Errorf("Function serialization should be stable: %v != %v", key1, key2)
	}

	funcPrefix := joinWithSeparator("GetWithFunc", "func") + ":"
	if !strings.HasPrefix(key1, funcPrefix) {
		t.Errorf("Function serialization should use func: prefix with pointer format, got: %v", key1)
	}
}

func TestDefaultKeySerializer_Pointers(t *testing.T) {
	serializer := NewDefaultKeySerializer()

	value := 42
	ptr := &value

	tests := []struct {
		name   string
		method string
		args   []any
		want   string
	}{
		{
			name:   "non-nil pointer",
			method: "GetByPtr",
			args:   []any{ptr},
			want:   joinWithSeparator("GetByPtr", "42"),
		},
		{
			name:   "nil pointer",
			method: "GetByPtr",
			args:   []any{(*int)(nil)},
			want:   joinWithSeparator("GetByPtr", "nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializer.SerializeKey(tt.method, tt.args...)
			if got != tt.want {
				t.Errorf("SerializeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultKeySerializer_Stability(t *testing.T) {
	serializer := NewDefaultKeySerializer()

	// Test that the same arguments produce the same key across multiple calls
	args := []any{1, "hello", []int{1, 2, 3}, map[string]int{"a": 1, "b": 2}}

	key1 := serializer.SerializeKey("TestMethod", args...)
	key2 := serializer.SerializeKey("TestMethod", args...)

	if key1 != key2 {
		t.Errorf("Key serialization should be stable across runs: %v != %v", key1, key2)
	}
}

func TestDefaultKeySerializer_JSONFallback(t *testing.T) {
	serializer := NewDefaultKeySerializer()

	// Test with a channel that should trigger JSON fallback
	ch := make(chan int)
	key := serializer.SerializeKey("GetWithChannel", ch)

	// Channel should be serialized with chan: prefix and pointer
	channelPrefix := joinWithSeparator("GetWithChannel", "chan") + ":"
	if !containsPrefix(key, channelPrefix) {
		t.Errorf("Channel should be serialized with chan: prefix, got: %v", key)
	}
}

// Helper function to check if string contains prefix
func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// Load test fixtures from JSON file (commented out since testsupport is not fully implemented)
func loadTestFixtures(t *testing.T) TestFixtures {
	t.Helper()

	filename := filepath.Join("testdata", "key_serializer_scenarios.json")
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read fixture file: %v", err)
	}

	var fixtures TestFixtures
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("Failed to unmarshal fixture data: %v", err)
	}

	return fixtures
}

func BenchmarkDefaultKeySerializer(b *testing.B) {
	serializer := NewDefaultKeySerializer()
	args := []any{1, "benchmark", []int{1, 2, 3}, map[string]int{"test": 1}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serializer.SerializeKey("BenchmarkMethod", args...)
	}
}
