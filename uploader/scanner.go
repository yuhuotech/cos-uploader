package uploader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hmw/cos-uploader/config"
	"github.com/hmw/cos-uploader/logger"
)

// ScanResult 扫描结果统计
type ScanResult struct {
	TotalFiles       int64
	TotalSize        int64
	FilesToUpload    map[string]*FileEntry
	FilesSkipped     int64
	Duration         string
	LocalIndex       *FileIndex
	RemoteIndex      *FileIndex
}

// DirectoryScanner 目录扫描器
type DirectoryScanner struct {
	logger        *logger.Logger
	hasher        *FileHasher
	projectConfig config.ProjectConfig
	indexManager  *IndexManager

	// 进度统计
	filesScanned int64
	totalSize    int64
	mu            sync.Mutex
}

// NewDirectoryScanner 创建目录扫描器
func NewDirectoryScanner(
	projectConfig config.ProjectConfig,
	indexManager *IndexManager,
	log *logger.Logger,
) *DirectoryScanner {
	return &DirectoryScanner{
		logger:        log,
		hasher:        NewFileHasher(),
		projectConfig: projectConfig,
		indexManager:  indexManager,
	}
}

// ScanDirectories 扫描所有监听目录并生成本地索引
func (ds *DirectoryScanner) ScanDirectories() (*FileIndex, error) {
	localIndex := NewFileIndex()

	// 递归扫描所有目录
	for _, dir := range ds.projectConfig.Directories {
		ds.logger.Info("Scanning directory", "path", dir, "project", ds.projectConfig.Name)

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				ds.logger.Warn("Error accessing path", "path", path, "error", err)
				return nil // 继续扫描其他文件
			}

			// 跳过目录
			if info.IsDir() {
				return nil
			}

			// 跳过隐藏文件和临时文件
			if strings.HasPrefix(info.Name(), ".") || strings.HasSuffix(info.Name(), ".tmp") {
				return nil
			}

			// 计算文件 MD5
			hash, size, err := ds.hasher.ComputeMD5(path)
			if err != nil {
				ds.logger.Warn("Failed to compute hash", "file", path, "error", err)
				return nil // 继续扫描其他文件
			}

			// 计算相对路径和远程路径
			relPath, _ := filepath.Rel(dir, path)
			remotePath := ds.projectConfig.COSConfig.PathPrefix + relPath

			// 标准化路径分隔符（Windows 使用 \，需要转换为 /）
			relPath = filepath.ToSlash(relPath)

			// 添加到本地索引
			localIndex.AddEntry(path, hash, size, remotePath)

			// 更新统计
			atomic.AddInt64(&ds.filesScanned, 1)
			atomic.AddInt64(&ds.totalSize, size)

			return nil
		})

		if err != nil {
			ds.logger.Error("Error scanning directory", "path", dir, "error", err)
			return nil, fmt.Errorf("failed to scan directory %s: %w", dir, err)
		}
	}

	ds.logger.Info("Directory scan completed",
		"project", ds.projectConfig.Name,
		"files", ds.filesScanned,
		"size", FormatBytes(ds.totalSize))

	return localIndex, nil
}

// AnalyzeForUpload 分析本地和远程索引，确定需要上传的文件
func (ds *DirectoryScanner) AnalyzeForUpload(localIdx, remoteIdx *FileIndex) (map[string]*FileEntry, int64) {
	needsUpload, skipped := CompareIndices(localIdx, remoteIdx)

	var uploadSize int64
	for _, entry := range needsUpload {
		uploadSize += entry.Size
	}

	ds.logger.Info("Upload analysis completed",
		"project", ds.projectConfig.Name,
		"files_to_upload", len(needsUpload),
		"files_skipped", skipped,
		"upload_size", FormatBytes(uploadSize))

	return needsUpload, int64(skipped)
}

// FormatBytes 将字节数格式化为可读的大小
func FormatBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(bytes)

	for _, unit := range units {
		if size < 1024 {
			return fmt.Sprintf("%.2f %s", size, unit)
		}
		size /= 1024
	}

	return fmt.Sprintf("%.2f TB", size)
}

// GetProgressCallback 获取进度回调函数
func (ds *DirectoryScanner) GetProgressCallback() func(current, total int64) {
	return func(current, total int64) {
		if total > 0 {
			percentage := float64(current) * 100 / float64(total)
			ds.logger.Debug("Upload progress",
				"project", ds.projectConfig.Name,
				"current", current,
				"total", total,
				"percentage", fmt.Sprintf("%.1f%%", percentage))
		}
	}
}
