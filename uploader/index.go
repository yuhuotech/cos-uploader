package uploader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/hmw/cos-uploader/config"
	"github.com/hmw/cos-uploader/logger"
	cos "github.com/tencentyun/cos-go-sdk-v5"
)

// FileEntry 索引中的文件条目
type FileEntry struct {
	Size           int64  `json:"size"`           // 文件大小（字节）
	Hash           string `json:"hash"`           // 文件 MD5 哈希值
	UploadedTime   string `json:"uploaded_time"`  // 上传时间戳
	RemotePath     string `json:"remote_path"`    // 远程路径
}

// FileIndex 本地或远程文件索引
type FileIndex struct {
	Version   string                `json:"version"`    // 索引版本
	Timestamp string                `json:"timestamp"`  // 索引生成时间
	Files     map[string]*FileEntry `json:"files"`      // 本地路径 -> 文件条目
}

// IndexManager 索引管理器
type IndexManager struct {
	logger    *logger.Logger
	cosClient *cos.Client
	cosConfig *config.COSConfig
}

// NewIndexManager 创建索引管理器
func NewIndexManager(cosClient *cos.Client, cosConfig *config.COSConfig, log *logger.Logger) *IndexManager {
	return &IndexManager{
		logger:    log,
		cosClient: cosClient,
		cosConfig: cosConfig,
	}
}

// NewFileIndex 创建新的空索引
func NewFileIndex() *FileIndex {
	return &FileIndex{
		Version:   "1.0",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Files:     make(map[string]*FileEntry),
	}
}

// AddEntry 向索引添加条目
func (idx *FileIndex) AddEntry(localPath, hash string, size int64, remotePath string) {
	idx.Files[localPath] = &FileEntry{
		Size:         size,
		Hash:         hash,
		UploadedTime: time.Now().UTC().Format(time.RFC3339),
		RemotePath:   remotePath,
	}
}

// GetEntry 从索引获取条目
func (idx *FileIndex) GetEntry(localPath string) *FileEntry {
	return idx.Files[localPath]
}

// SaveToFile 保存本地索引文件
func (idx *FileIndex) SaveToFile(filePath string) error {
	// 创建目录
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// 转换为 JSON
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// LoadFromFile 从本地文件加载索引
func LoadFileIndexFromFile(filePath string) (*FileIndex, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，返回空索引
			return NewFileIndex(), nil
		}
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var idx FileIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return &idx, nil
}

// GetLocalIndexPath 获取本地索引文件路径
func GetLocalIndexPath(projectName string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	return filepath.Join(homeDir, ".cos-uploader", projectName, "local_index.json")
}

// DownloadRemoteIndex 从 COS 下载远程索引
func (im *IndexManager) DownloadRemoteIndex(ctx context.Context, projectName string) (*FileIndex, error) {
	// 远程索引路径
	remoteIndexPath := im.cosConfig.PathPrefix + ".cos-uploader/" + projectName + "/remote_index.json"

	// 尝试下载
	resp, err := im.cosClient.Object.Get(ctx, remoteIndexPath, nil)
	if err != nil {
		// 如果文件不存在，返回新的空索引
		if e, ok := err.(*cos.ErrorResponse); ok && e.Code == "NoSuchKey" {
			im.logger.Info("Remote index not found, creating new one", "project", projectName)
			return NewFileIndex(), nil
		}
		// 其他错误
		im.logger.Warn("Failed to download remote index", "project", projectName, "error", err)
		return NewFileIndex(), nil // 降级处理，继续上传
	}
	defer resp.Body.Close()

	// 读取响应体
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote index response: %w", err)
	}

	// 解析 JSON
	var idx FileIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal remote index: %w", err)
	}

	im.logger.Info("Remote index downloaded", "project", projectName, "entries", len(idx.Files))
	return &idx, nil
}

// UploadRemoteIndex 上传远程索引到 COS
func (im *IndexManager) UploadRemoteIndex(ctx context.Context, idx *FileIndex, projectName string) error {
	// 转换为 JSON
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// 远程索引路径
	remoteIndexPath := im.cosConfig.PathPrefix + ".cos-uploader/" + projectName + "/remote_index.json"

	// 上传到 COS
	_, err = im.cosClient.Object.Put(ctx, remoteIndexPath, bytes.NewReader(data), nil)
	if err != nil {
		return fmt.Errorf("failed to upload remote index: %w", err)
	}

	im.logger.Info("Remote index uploaded", "project", projectName, "entries", len(idx.Files))
	return nil
}

// CompareWithRemote 对比本地和远程索引，返回需要上传的文件
// 返回值: 需要上传的文件 map，已跳过的数量
func CompareIndices(localIdx, remoteIdx *FileIndex) (map[string]*FileEntry, int) {
	needsUpload := make(map[string]*FileEntry)
	skipped := 0

	for localPath, localEntry := range localIdx.Files {
		remoteEntry, exists := remoteIdx.Files[localPath]

		if !exists {
			// 文件不在远程索引中，需要上传
			needsUpload[localPath] = localEntry
		} else if remoteEntry.Hash != localEntry.Hash {
			// 文件存在但哈希不同，需要重新上传
			needsUpload[localPath] = localEntry
		} else {
			// 文件已存在且哈希相同，跳过
			skipped++
		}
	}

	return needsUpload, skipped
}

// UpdateRemoteIndexWithUploads 使用上传结果更新远程索引
func UpdateRemoteIndexWithUploads(remoteIdx *FileIndex, uploads map[string]*FileEntry) {
	for localPath, entry := range uploads {
		// 用本次上传的信息更新远程索引
		remoteIdx.Files[localPath] = &FileEntry{
			Size:         entry.Size,
			Hash:         entry.Hash,
			UploadedTime: time.Now().UTC().Format(time.RFC3339),
			RemotePath:   entry.RemotePath,
		}
	}
	// 更新时间戳
	remoteIdx.Timestamp = time.Now().UTC().Format(time.RFC3339)
}
