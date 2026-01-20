package uploader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFileIndex(t *testing.T) {
	idx := NewFileIndex()

	if idx == nil {
		t.Fatal("NewFileIndex returned nil")
	}

	if idx.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", idx.Version)
	}

	if idx.Files == nil {
		t.Fatal("Files map is nil")
	}

	if len(idx.Files) != 0 {
		t.Errorf("Expected empty Files map, got %d entries", len(idx.Files))
	}
}

func TestAddEntry(t *testing.T) {
	idx := NewFileIndex()

	localPath := "/path/to/file.txt"
	hash := "abc123def456"
	size := int64(1024)
	remotePath := "prefix/file.txt"

	idx.AddEntry(localPath, hash, size, remotePath)

	entry, ok := idx.Files[localPath]
	if !ok {
		t.Fatal("Entry not found in index")
	}

	if entry.Hash != hash {
		t.Errorf("Expected hash %s, got %s", hash, entry.Hash)
	}

	if entry.Size != size {
		t.Errorf("Expected size %d, got %d", size, entry.Size)
	}

	if entry.RemotePath != remotePath {
		t.Errorf("Expected remotePath %s, got %s", remotePath, entry.RemotePath)
	}

	if entry.UploadedTime == "" {
		t.Fatal("UploadedTime should not be empty")
	}
}

func TestGetEntry(t *testing.T) {
	idx := NewFileIndex()
	localPath := "/path/to/file.txt"
	hash := "abc123"
	size := int64(512)
	remotePath := "prefix/file.txt"

	idx.AddEntry(localPath, hash, size, remotePath)

	entry := idx.GetEntry(localPath)
	if entry == nil {
		t.Fatal("GetEntry returned nil for existing entry")
	}

	if entry.Hash != hash {
		t.Errorf("Expected hash %s, got %s", hash, entry.Hash)
	}

	// Test non-existent entry
	nonExistent := idx.GetEntry("/non/existent/path")
	if nonExistent != nil {
		t.Fatal("GetEntry should return nil for non-existent entry")
	}
}

func TestSaveToFile(t *testing.T) {
	idx := NewFileIndex()
	idx.AddEntry("/file1.txt", "hash1", 100, "remote1.txt")
	idx.AddEntry("/file2.txt", "hash2", 200, "remote2.txt")

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_index.json")

	err := idx.SaveToFile(filePath)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("Index file was not created: %v", err)
	}

	// Verify content can be read
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read index file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Index file is empty")
	}
}

func TestLoadFileIndexFromFile(t *testing.T) {
	// Create and save an index
	originalIdx := NewFileIndex()
	originalIdx.AddEntry("/file1.txt", "hash1", 100, "remote1.txt")
	originalIdx.AddEntry("/file2.txt", "hash2", 200, "remote2.txt")

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_index.json")

	err := originalIdx.SaveToFile(filePath)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load it back
	loadedIdx, err := LoadFileIndexFromFile(filePath)
	if err != nil {
		t.Fatalf("LoadFileIndexFromFile failed: %v", err)
	}

	if loadedIdx == nil {
		t.Fatal("LoadFileIndexFromFile returned nil")
	}

	if len(loadedIdx.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(loadedIdx.Files))
	}

	// Verify entries
	entry1 := loadedIdx.GetEntry("/file1.txt")
	if entry1 == nil || entry1.Hash != "hash1" {
		t.Fatal("File1 entry not found or hash mismatch")
	}

	entry2 := loadedIdx.GetEntry("/file2.txt")
	if entry2 == nil || entry2.Hash != "hash2" {
		t.Fatal("File2 entry not found or hash mismatch")
	}
}

func TestLoadFileIndexFromFile_NonExistent(t *testing.T) {
	nonExistentPath := "/path/that/does/not/exist/index.json"

	idx, err := LoadFileIndexFromFile(nonExistentPath)
	if err != nil {
		t.Fatalf("LoadFileIndexFromFile should not error for non-existent file: %v", err)
	}

	if idx == nil {
		t.Fatal("LoadFileIndexFromFile should return empty index for non-existent file")
	}

	if len(idx.Files) != 0 {
		t.Errorf("Expected empty index, got %d entries", len(idx.Files))
	}
}

func TestCompareIndices(t *testing.T) {
	localIdx := NewFileIndex()
	localIdx.AddEntry("/file1.txt", "hash1", 100, "prefix/file1.txt")
	localIdx.AddEntry("/file2.txt", "hash2", 200, "prefix/file2.txt")
	localIdx.AddEntry("/file3.txt", "hash3_new", 300, "prefix/file3.txt")

	remoteIdx := NewFileIndex()
	remoteIdx.AddEntry("/file2.txt", "hash2", 200, "prefix/file2.txt")
	remoteIdx.AddEntry("/file3.txt", "hash3_old", 300, "prefix/file3.txt")

	needsUpload, skipped := CompareIndices(localIdx, remoteIdx)

	// file1 should be uploaded (not in remote)
	// file2 should be skipped (exists with same hash)
	// file3 should be uploaded (exists but hash differs)

	if len(needsUpload) != 2 {
		t.Errorf("Expected 2 files to upload, got %d", len(needsUpload))
	}

	if skipped != 1 {
		t.Errorf("Expected 1 file skipped, got %d", skipped)
	}

	if _, ok := needsUpload["/file1.txt"]; !ok {
		t.Fatal("file1.txt should be in needsUpload")
	}

	if _, ok := needsUpload["/file3.txt"]; !ok {
		t.Fatal("file3.txt should be in needsUpload (hash mismatch)")
	}

	if _, ok := needsUpload["/file2.txt"]; ok {
		t.Fatal("file2.txt should not be in needsUpload (hash matches)")
	}
}

func TestUpdateRemoteIndexWithUploads(t *testing.T) {
	remoteIdx := NewFileIndex()

	// Simulate uploaded files
	uploads := make(map[string]*FileEntry)
	uploads["/file1.txt"] = &FileEntry{
		Size:       100,
		Hash:       "newhash1",
		RemotePath: "prefix/file1.txt",
	}
	uploads["/file2.txt"] = &FileEntry{
		Size:       200,
		Hash:       "newhash2",
		RemotePath: "prefix/file2.txt",
	}

	UpdateRemoteIndexWithUploads(remoteIdx, uploads)

	if len(remoteIdx.Files) != 2 {
		t.Errorf("Expected 2 entries in remote index, got %d", len(remoteIdx.Files))
	}

	entry1 := remoteIdx.GetEntry("/file1.txt")
	if entry1 == nil || entry1.Hash != "newhash1" {
		t.Fatal("File1 entry not found or hash mismatch")
	}

	// Timestamp should be a valid RFC3339 format
	if remoteIdx.Timestamp == "" {
		t.Fatal("Timestamp should not be empty")
	}

	// Verify both entries have upload times
	for _, entry := range remoteIdx.Files {
		if entry.UploadedTime == "" {
			t.Fatal("UploadedTime should be set for all entries")
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string // Prefix check since float formatting might vary slightly
	}{
		{0, "0.00 B"},
		{512, "512.00 B"},
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}

	for _, test := range tests {
		result := FormatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("FormatBytes(%d): expected %s, got %s", test.bytes, test.expected, result)
		}
	}
}

func TestGetLocalIndexPath(t *testing.T) {
	projectName := "test-project"
	path := GetLocalIndexPath(projectName)

	if path == "" {
		t.Fatal("GetLocalIndexPath returned empty path")
	}

	// Path should contain project name and local_index.json
	if !strings.Contains(path, projectName) {
		t.Errorf("Path should contain project name: %s", path)
	}

	if !strings.Contains(path, "local_index.json") {
		t.Errorf("Path should contain local_index.json: %s", path)
	}

	if !strings.Contains(path, ".cos-uploader") {
		t.Errorf("Path should contain .cos-uploader: %s", path)
	}
}
