package uploader

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

// FileHasher 文件哈希计算器
type FileHasher struct {
	bufferSize int // 读取缓冲区大小，默认 32MB
}

// NewFileHasher 创建文件哈希计算器
func NewFileHasher() *FileHasher {
	return &FileHasher{
		bufferSize: 32 * 1024 * 1024, // 32MB 缓冲
	}
}

// ComputeMD5 计算文件的 MD5 哈希
// 使用流式计算，支持大文件
func (h *FileHasher) ComputeMD5(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return "", 0, fmt.Errorf("failed to stat file: %w", err)
	}
	fileSize := fileInfo.Size()

	// 计算 MD5
	hash := md5.New()
	buffer := make([]byte, h.bufferSize)

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", 0, fmt.Errorf("failed to read file: %w", err)
		}
	}

	// 返回十六进制格式的哈希值
	return fmt.Sprintf("%x", hash.Sum(nil)), fileSize, nil
}

// ComputeMD5Batch 批量计算多个文件的 MD5
// 返回 map[filePath]hash 和任何错误
func (h *FileHasher) ComputeMD5Batch(filePaths []string) (map[string]string, error) {
	results := make(map[string]string)

	for _, filePath := range filePaths {
		hash, _, err := h.ComputeMD5(filePath)
		if err != nil {
			// 记录错误但继续处理其他文件
			// 由调用者决定是否中断
			results[filePath] = ""
			continue
		}
		results[filePath] = hash
	}

	return results, nil
}
