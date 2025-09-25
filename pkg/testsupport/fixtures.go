package testsupport

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// LoadFixture loads test data from a fixture file.
// The path is relative to the test package directory.
func LoadFixture(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to load fixture from %s: %v", path, err)
	}

	return data
}

// LoadFixtureJSON loads JSON test data from a fixture file and unmarshals it.
// The path is relative to the test package directory.
func LoadFixtureJSON(t *testing.T, path string, dest interface{}) {
	t.Helper()

	data := LoadFixture(t, path)
	if err := json.Unmarshal(data, dest); err != nil {
		t.Fatalf("failed to unmarshal JSON fixture from %s: %v", path, err)
	}
}

// LoadGolden loads expected test output from a golden file.
// The path is relative to the test package directory.
func LoadGolden(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to load golden file from %s: %v", path, err)
	}

	return data
}

// WriteGolden writes test output to a golden file.
// This should typically only be called when updating golden files.
// The path is relative to the test package directory.
func WriteGolden(t *testing.T, path string, data []byte) {
	t.Helper()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write golden file to %s: %v", path, err)
	}
}

// WriteGoldenJSON writes JSON test output to a golden file.
// This should typically only be called when updating golden files.
func WriteGoldenJSON(t *testing.T, path string, data interface{}) {
	t.Helper()

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON for golden file %s: %v", path, err)
	}

	WriteGolden(t, path, jsonData)
}

// CompareWithGolden compares actual data with expected data from a golden file.
// If the golden file doesn't exist, it creates one with the actual data.
func CompareWithGolden(t *testing.T, path string, actual []byte) {
	t.Helper()

	expected, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Logf("Golden file %s does not exist, creating it", path)
			WriteGolden(t, path, actual)
			return
		}
		t.Fatalf("failed to read golden file %s: %v", path, err)
	}

	if string(actual) != string(expected) {
		t.Errorf("output mismatch for %s:\nExpected:\n%s\nActual:\n%s", path, expected, actual)
	}
}

// LoadReader creates an io.Reader from fixture data.
// Useful for testing functions that accept readers.
func LoadReader(t *testing.T, path string) io.Reader {
	t.Helper()

	data := LoadFixture(t, path)
	return strings.NewReader(string(data))
}

// TempFile creates a temporary file with the given content for testing.
// The caller is responsible for cleaning up the file.
func TempFile(t *testing.T, content []byte) string {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := tmpfile.Write(content); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		t.Fatalf("failed to write to temp file: %v", err)
	}

	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		t.Fatalf("failed to close temp file: %v", err)
	}

	return tmpfile.Name()
}

// TempDir creates a temporary directory for testing.
// The caller is responsible for cleaning up the directory.
func TempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	return dir
}

// FixturePath constructs a path to a fixture file relative to the testdata directory.
func FixturePath(filename string) string {
	return filepath.Join("testdata", filename)
}

// GoldenPath constructs a path to a golden file relative to the testdata directory.
func GoldenPath(filename string) string {
	return filepath.Join("testdata", "golden", filename)
}
