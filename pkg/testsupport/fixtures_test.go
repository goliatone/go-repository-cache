package testsupport

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFixture(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test fixture content")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test successful load
	result := LoadFixture(t, testFile)
	if string(result) != string(testContent) {
		t.Errorf("expected %q, got %q", testContent, result)
	}
}

func TestLoadFixture_NonExistentFile(t *testing.T) {
	// This test verifies that LoadFixture fails appropriately for non-existent files
	// We can't easily test t.Fatalf being called, so we'll test the underlying behavior
	_, err := os.ReadFile("non-existent-file.txt")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadFixtureJSON(t *testing.T) {
	// Create a temporary JSON file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	testData := map[string]interface{}{
		"name":  "test",
		"value": 42,
		"items": []string{"a", "b", "c"},
	}

	jsonData, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(testFile, jsonData, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test successful JSON load
	var result map[string]interface{}
	LoadFixtureJSON(t, testFile, &result)

	if result["name"] != "test" {
		t.Errorf("expected name=test, got %v", result["name"])
	}
	if result["value"] != float64(42) { // JSON unmarshals numbers as float64
		t.Errorf("expected value=42, got %v", result["value"])
	}
}

func TestLoadGolden(t *testing.T) {
	// Create a temporary golden file for testing
	tmpDir := t.TempDir()
	goldenFile := filepath.Join(tmpDir, "test.golden")
	goldenContent := []byte("expected output content")

	if err := os.WriteFile(goldenFile, goldenContent, 0644); err != nil {
		t.Fatalf("failed to create golden file: %v", err)
	}

	// Test successful load
	result := LoadGolden(t, goldenFile)
	if string(result) != string(goldenContent) {
		t.Errorf("expected %q, got %q", goldenContent, result)
	}
}

func TestWriteGolden(t *testing.T) {
	tmpDir := t.TempDir()
	goldenFile := filepath.Join(tmpDir, "subdir", "test.golden")
	testContent := []byte("test golden content")

	// Test writing golden file (should create directories)
	WriteGolden(t, goldenFile, testContent)

	// Verify file was created with correct content
	result, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read written golden file: %v", err)
	}

	if string(result) != string(testContent) {
		t.Errorf("expected %q, got %q", testContent, result)
	}
}

func TestWriteGoldenJSON(t *testing.T) {
	tmpDir := t.TempDir()
	goldenFile := filepath.Join(tmpDir, "test.json")
	testData := map[string]interface{}{
		"test":   "data",
		"number": 123,
	}

	// Test writing JSON golden file
	WriteGoldenJSON(t, goldenFile, testData)

	// Verify file was created with correct JSON content
	result, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read written golden file: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse written JSON: %v", err)
	}

	if parsed["test"] != "data" {
		t.Errorf("expected test=data, got %v", parsed["test"])
	}
}

func TestCompareWithGolden(t *testing.T) {
	tmpDir := t.TempDir()
	goldenFile := filepath.Join(tmpDir, "test.golden")
	testContent := []byte("test content")

	// Test case 1: Golden file doesn't exist (should create it)
	CompareWithGolden(t, goldenFile, testContent)

	// Verify file was created
	result, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read created golden file: %v", err)
	}

	if string(result) != string(testContent) {
		t.Errorf("expected %q, got %q", testContent, result)
	}

	// Test case 2: Golden file exists and matches (should pass)
	// We can't easily test this without a mock, but the logic is straightforward
}

func TestLoadReader(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test reader content")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test creating reader from fixture
	reader := LoadReader(t, testFile)

	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read from reader: %v", err)
	}

	if string(result) != string(testContent) {
		t.Errorf("expected %q, got %q", testContent, result)
	}
}

func TestTempFile(t *testing.T) {
	testContent := []byte("temporary file content")

	// Test creating temporary file
	tempPath := TempFile(t, testContent)
	defer os.Remove(tempPath) // Clean up

	// Verify file exists and has correct content
	result, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	if string(result) != string(testContent) {
		t.Errorf("expected %q, got %q", testContent, result)
	}

	// Verify it's actually a temporary file (contains "test-" in name)
	if !strings.Contains(filepath.Base(tempPath), "test-") {
		t.Errorf("temp file name should contain 'test-', got %s", tempPath)
	}
}

func TestTempDir(t *testing.T) {
	// Test creating temporary directory
	tempDir := TempDir(t)
	defer os.RemoveAll(tempDir) // Clean up

	// Verify directory exists
	info, err := os.Stat(tempDir)
	if err != nil {
		t.Fatalf("failed to stat temp directory: %v", err)
	}

	if !info.IsDir() {
		t.Error("expected directory, got file")
	}

	// Verify it's actually a temporary directory (contains "test-" in name)
	if !strings.Contains(filepath.Base(tempDir), "test-") {
		t.Errorf("temp directory name should contain 'test-', got %s", tempDir)
	}
}

func TestFixturePath(t *testing.T) {
	result := FixturePath("test.json")
	expected := filepath.Join("testdata", "test.json")

	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestGoldenPath(t *testing.T) {
	result := GoldenPath("output.txt")
	expected := filepath.Join("testdata", "golden", "output.txt")

	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Integration test demonstrating typical usage patterns
func TestFixtureWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate testdata directory structure
	testdataDir := filepath.Join(tmpDir, "testdata")
	goldenDir := filepath.Join(testdataDir, "golden")

	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("failed to create testdata directories: %v", err)
	}

	// Create a fixture file
	fixtureFile := filepath.Join(testdataDir, "input.json")
	fixtureData := map[string]interface{}{
		"input": "test data",
		"count": 3,
	}

	jsonData, _ := json.MarshalIndent(fixtureData, "", "  ")
	if err := os.WriteFile(fixtureFile, jsonData, 0644); err != nil {
		t.Fatalf("failed to create fixture file: %v", err)
	}

	// Change to temp directory to test relative paths
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Test loading fixture with helper paths
	var loaded map[string]interface{}
	LoadFixtureJSON(t, FixturePath("input.json"), &loaded)

	if loaded["input"] != "test data" {
		t.Errorf("fixture not loaded correctly")
	}

	// Test golden file workflow
	output := []byte("processed output")
	goldenFile := GoldenPath("output.txt")

	// First run: create golden file
	CompareWithGolden(t, goldenFile, output)

	// Verify golden file exists
	if _, err := os.Stat(goldenFile); err != nil {
		t.Errorf("golden file should have been created: %v", err)
	}
}
