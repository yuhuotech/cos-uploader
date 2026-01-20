package uploader

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileHasher(t *testing.T) {
	hasher := NewFileHasher()

	if hasher == nil {
		t.Fatal("NewFileHasher returned nil")
	}

	if hasher.bufferSize != 32*1024*1024 {
		t.Errorf("Expected buffer size 32MB, got %d", hasher.bufferSize)
	}
}

func TestComputeMD5(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file with known content
	testContent := "Hello, World!"
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hasher := NewFileHasher()
	hash, size, err := hasher.ComputeMD5(testFile)

	if err != nil {
		t.Fatalf("ComputeMD5 failed: %v", err)
	}

	if size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), size)
	}

	// Calculate expected MD5
	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte(testContent)))
	if hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, hash)
	}
}

func TestComputeMD5_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// Create a file larger than typical buffer for testing streaming (5MB)
	largeContent := make([]byte, 5*1024*1024) // 5MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	err := os.WriteFile(testFile, largeContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	hasher := NewFileHasher()
	hash, size, err := hasher.ComputeMD5(testFile)

	if err != nil {
		t.Fatalf("ComputeMD5 failed for large file: %v", err)
	}

	if size != int64(len(largeContent)) {
		t.Errorf("Expected size %d, got %d", len(largeContent), size)
	}

	if hash == "" {
		t.Fatal("Hash should not be empty")
	}

	if len(hash) != 32 {
		t.Errorf("MD5 hash should be 32 characters, got %d", len(hash))
	}
}

func TestComputeMD5_NonExistent(t *testing.T) {
	hasher := NewFileHasher()
	_, _, err := hasher.ComputeMD5("/path/that/does/not/exist.txt")

	if err == nil {
		t.Fatal("ComputeMD5 should error for non-existent file")
	}
}

func TestComputeMD5_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	err := os.WriteFile(testFile, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}

	hasher := NewFileHasher()
	hash, size, err := hasher.ComputeMD5(testFile)

	if err != nil {
		t.Fatalf("ComputeMD5 failed for empty file: %v", err)
	}

	if size != 0 {
		t.Errorf("Expected size 0 for empty file, got %d", size)
	}

	// MD5 of empty string
	expectedHash := "d41d8cd98f00b204e9800998ecf8427e"
	if hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, hash)
	}
}

func TestComputeMD5Batch(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple test files
	files := []string{
		filepath.Join(tmpDir, "file1.txt"),
		filepath.Join(tmpDir, "file2.txt"),
		filepath.Join(tmpDir, "file3.txt"),
	}

	for i, file := range files {
		content := fmt.Sprintf("Content %d", i)
		err := os.WriteFile(file, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	hasher := NewFileHasher()
	results, err := hasher.ComputeMD5Batch(files)

	if err != nil {
		t.Fatalf("ComputeMD5Batch failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Verify all files have hashes
	for _, file := range files {
		hash, ok := results[file]
		if !ok {
			t.Errorf("File %s not in results", file)
		}

		if hash == "" {
			t.Errorf("Hash for %s should not be empty", file)
		}
	}
}

func TestComputeMD5Batch_WithError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some valid files and include non-existent ones
	files := []string{
		filepath.Join(tmpDir, "file1.txt"),
		"/path/that/does/not/exist.txt",
		filepath.Join(tmpDir, "file2.txt"),
	}

	// Create the valid files
	err := os.WriteFile(files[0], []byte("Content 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(files[2], []byte("Content 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hasher := NewFileHasher()
	results, err := hasher.ComputeMD5Batch(files)

	if err != nil {
		t.Fatalf("ComputeMD5Batch should not error at batch level: %v", err)
	}

	// Should have entries for all files
	if len(results) != 3 {
		t.Errorf("Expected 3 results (including errors), got %d", len(results))
	}

	// Valid files should have hashes
	if hash := results[files[0]]; hash == "" {
		t.Fatal("Valid file should have hash")
	}

	// Invalid file should have empty hash
	if hash := results[files[1]]; hash != "" {
		t.Errorf("Invalid file should have empty hash, got %s", hash)
	}

	if hash := results[files[2]]; hash == "" {
		t.Fatal("Valid file should have hash")
	}
}

func TestComputeMD5_Consistency(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Consistent content"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hasher := NewFileHasher()

	// Compute MD5 multiple times, should get same result
	hash1, size1, err1 := hasher.ComputeMD5(testFile)
	hash2, size2, err2 := hasher.ComputeMD5(testFile)

	if err1 != nil || err2 != nil {
		t.Fatalf("ComputeMD5 failed: %v, %v", err1, err2)
	}

	if hash1 != hash2 {
		t.Errorf("MD5 should be consistent, got %s and %s", hash1, hash2)
	}

	if size1 != size2 {
		t.Errorf("Size should be consistent, got %d and %d", size1, size2)
	}
}

func TestComputeMD5_Different_Files(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	err := os.WriteFile(file1, []byte("Content A"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(file2, []byte("Content B"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hasher := NewFileHasher()
	hash1, _, _ := hasher.ComputeMD5(file1)
	hash2, _, _ := hasher.ComputeMD5(file2)

	if hash1 == hash2 {
		t.Fatal("Different files should have different hashes")
	}
}
