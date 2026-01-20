package uploader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hmw/cos-uploader/config"
	"github.com/hmw/cos-uploader/logger"
)

func TestNewDirectoryScanner(t *testing.T) {
	log := logger.NewLogger()
	defer log.Sync()

	projectConfig := config.ProjectConfig{
		Name:        "test-project",
		Directories: []string{"/path/to/dir"},
		COSConfig: config.COSConfig{
			PathPrefix: "prefix/",
		},
	}

	indexManager := NewIndexManager(nil, &projectConfig.COSConfig, log)
	scanner := NewDirectoryScanner(projectConfig, indexManager, log)

	if scanner == nil {
		t.Fatal("NewDirectoryScanner returned nil")
	}

	if scanner.projectConfig.Name != "test-project" {
		t.Errorf("Expected project name test-project, got %s", scanner.projectConfig.Name)
	}

	if scanner.filesScanned != 0 {
		t.Errorf("Expected 0 files scanned initially, got %d", scanner.filesScanned)
	}

	if scanner.totalSize != 0 {
		t.Errorf("Expected 0 total size initially, got %d", scanner.totalSize)
	}
}

func TestScanDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.NewLogger()
	defer log.Sync()

	// Create test file structure
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "subdir", "file2.txt")
	os.MkdirAll(filepath.Dir(file2), 0755)

	err := os.WriteFile(file1, []byte("Content 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(file2, []byte("Content 22"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create hidden file (should be skipped)
	hiddenFile := filepath.Join(tmpDir, ".hidden")
	err = os.WriteFile(hiddenFile, []byte("Hidden"), 0644)
	if err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}

	// Create tmp file (should be skipped)
	tmpFile := filepath.Join(tmpDir, "temp.tmp")
	err = os.WriteFile(tmpFile, []byte("Temp"), 0644)
	if err != nil {
		t.Fatalf("Failed to create tmp file: %v", err)
	}

	projectConfig := config.ProjectConfig{
		Name:        "test-project",
		Directories: []string{tmpDir},
		COSConfig: config.COSConfig{
			PathPrefix: "prefix/",
		},
	}

	indexManager := NewIndexManager(nil, &projectConfig.COSConfig, log)
	scanner := NewDirectoryScanner(projectConfig, indexManager, log)

	localIndex, err := scanner.ScanDirectories()

	if err != nil {
		t.Fatalf("ScanDirectories failed: %v", err)
	}

	if localIndex == nil {
		t.Fatal("ScanDirectories returned nil index")
	}

	// Should have scanned 2 files (hidden and tmp should be skipped)
	if scanner.filesScanned != 2 {
		t.Errorf("Expected 2 files scanned, got %d", scanner.filesScanned)
	}

	// Verify files are in index
	if localIndex.GetEntry(file1) == nil {
		t.Fatal("file1.txt not found in index")
	}

	if localIndex.GetEntry(file2) == nil {
		t.Fatal("file2.txt not found in index")
	}

	// Hidden and tmp files should not be in index
	if localIndex.GetEntry(hiddenFile) != nil {
		t.Fatal("Hidden file should not be in index")
	}

	if localIndex.GetEntry(tmpFile) != nil {
		t.Fatal("Tmp file should not be in index")
	}

	// Verify file entries have correct hash and size
	entry1 := localIndex.GetEntry(file1)
	if entry1.Size != 9 {
		t.Errorf("Expected size 9, got %d", entry1.Size)
	}

	if entry1.Hash == "" {
		t.Fatal("Hash should not be empty")
	}

	entry2 := localIndex.GetEntry(file2)
	if entry2.Size != 10 {
		t.Errorf("Expected size 10, got %d", entry2.Size)
	}
}

func TestScanDirectories_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.NewLogger()
	defer log.Sync()

	projectConfig := config.ProjectConfig{
		Name:        "test-project",
		Directories: []string{tmpDir},
		COSConfig: config.COSConfig{
			PathPrefix: "prefix/",
		},
	}

	indexManager := NewIndexManager(nil, &projectConfig.COSConfig, log)
	scanner := NewDirectoryScanner(projectConfig, indexManager, log)

	localIndex, err := scanner.ScanDirectories()

	if err != nil {
		t.Fatalf("ScanDirectories failed: %v", err)
	}

	if len(localIndex.Files) != 0 {
		t.Errorf("Expected empty index for empty directory, got %d entries", len(localIndex.Files))
	}

	if scanner.filesScanned != 0 {
		t.Errorf("Expected 0 files scanned, got %d", scanner.filesScanned)
	}
}

func TestAnalyzeForUpload(t *testing.T) {
	log := logger.NewLogger()
	defer log.Sync()

	projectConfig := config.ProjectConfig{
		Name:        "test-project",
		Directories: []string{"/path"},
		COSConfig: config.COSConfig{
			PathPrefix: "prefix/",
		},
	}

	indexManager := NewIndexManager(nil, &projectConfig.COSConfig, log)
	scanner := NewDirectoryScanner(projectConfig, indexManager, log)

	// Create local index with 3 files
	localIdx := NewFileIndex()
	localIdx.AddEntry("/file1.txt", "hash1", 100, "prefix/file1.txt")
	localIdx.AddEntry("/file2.txt", "hash2", 200, "prefix/file2.txt")
	localIdx.AddEntry("/file3.txt", "hash3", 300, "prefix/file3.txt")

	// Create remote index with 2 files (file2 is duplicate, file3 has different hash)
	remoteIdx := NewFileIndex()
	remoteIdx.AddEntry("/file2.txt", "hash2", 200, "prefix/file2.txt")
	remoteIdx.AddEntry("/file3.txt", "hash3_old", 300, "prefix/file3.txt")

	filesToUpload, skipped := scanner.AnalyzeForUpload(localIdx, remoteIdx)

	if len(filesToUpload) != 2 {
		t.Errorf("Expected 2 files to upload, got %d", len(filesToUpload))
	}

	if skipped != 1 {
		t.Errorf("Expected 1 file skipped, got %d", skipped)
	}

	if _, ok := filesToUpload["/file1.txt"]; !ok {
		t.Fatal("file1.txt should be in upload list")
	}

	if _, ok := filesToUpload["/file3.txt"]; !ok {
		t.Fatal("file3.txt should be in upload list (hash mismatch)")
	}

	if _, ok := filesToUpload["/file2.txt"]; ok {
		t.Fatal("file2.txt should not be in upload list")
	}
}

func TestFormatBytes_UnitConversion(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0.00 B"},
		{1, "1.00 B"},
		{512, "512.00 B"},
		{1023, "1023.00 B"},
		{1024, "1.00 KB"},
		{1024 * 512, "512.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 512, "512.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 512, "512.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}

	for _, test := range tests {
		result := FormatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("FormatBytes(%d): expected %s, got %s", test.bytes, test.expected, result)
		}
	}
}

func TestGetProgressCallback(t *testing.T) {
	log := logger.NewLogger()
	defer log.Sync()

	projectConfig := config.ProjectConfig{
		Name:        "test-project",
		Directories: []string{"/path"},
		COSConfig: config.COSConfig{
			PathPrefix: "prefix/",
		},
	}

	indexManager := NewIndexManager(nil, &projectConfig.COSConfig, log)
	scanner := NewDirectoryScanner(projectConfig, indexManager, log)

	callback := scanner.GetProgressCallback()

	if callback == nil {
		t.Fatal("GetProgressCallback returned nil")
	}

	// Should not panic when called
	callback(50, 100)
	callback(100, 100)
	callback(0, 100)
	callback(0, 0) // Edge case: total = 0
}

func TestScanDirectories_MultipleFolders(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	log := logger.NewLogger()
	defer log.Sync()

	// Create files in first directory
	file1 := filepath.Join(tmpDir1, "file1.txt")
	err := os.WriteFile(file1, []byte("Content 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create files in second directory
	file2 := filepath.Join(tmpDir2, "file2.txt")
	err = os.WriteFile(file2, []byte("Content 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	projectConfig := config.ProjectConfig{
		Name:        "test-project",
		Directories: []string{tmpDir1, tmpDir2},
		COSConfig: config.COSConfig{
			PathPrefix: "prefix/",
		},
	}

	indexManager := NewIndexManager(nil, &projectConfig.COSConfig, log)
	scanner := NewDirectoryScanner(projectConfig, indexManager, log)

	localIndex, err := scanner.ScanDirectories()

	if err != nil {
		t.Fatalf("ScanDirectories failed: %v", err)
	}

	// Should have scanned 2 files from both directories
	if scanner.filesScanned != 2 {
		t.Errorf("Expected 2 files scanned, got %d", scanner.filesScanned)
	}

	// Both files should be in the index
	if len(localIndex.Files) != 2 {
		t.Errorf("Expected 2 entries in index, got %d", len(localIndex.Files))
	}
}

func TestScanDirectories_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	log := logger.NewLogger()
	defer log.Sync()

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "level1", "level2", "level3")
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directories: %v", err)
	}

	deepFile := filepath.Join(nestedDir, "deep.txt")
	err = os.WriteFile(deepFile, []byte("Deep content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create deep file: %v", err)
	}

	projectConfig := config.ProjectConfig{
		Name:        "test-project",
		Directories: []string{tmpDir},
		COSConfig: config.COSConfig{
			PathPrefix: "prefix/",
		},
	}

	indexManager := NewIndexManager(nil, &projectConfig.COSConfig, log)
	scanner := NewDirectoryScanner(projectConfig, indexManager, log)

	localIndex, err := scanner.ScanDirectories()

	if err != nil {
		t.Fatalf("ScanDirectories failed: %v", err)
	}

	// Should find the deeply nested file
	if scanner.filesScanned != 1 {
		t.Errorf("Expected 1 file scanned, got %d", scanner.filesScanned)
	}

	if localIndex.GetEntry(deepFile) == nil {
		t.Fatal("Deep file not found in index")
	}
}
