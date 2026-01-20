package logger

import (
	"os"
	"testing"
)

func TestNewLogger(t *testing.T) {
	// 清理日志目录
	os.RemoveAll("logs")

	log := NewLogger()
	if log == nil {
		t.Fatal("Expected logger, got nil")
	}

	// 检查日志目录是否创建
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		t.Error("Logs directory was not created")
	}

	// 测试日志输出
	log.Info("Test info message", "key", "value")
	log.Error("Test error message", "key", "value")

	log.Sync()

	// 检查日志文件是否创建
	if _, err := os.Stat("logs/cos-uploader.log"); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	// 清理
	os.RemoveAll("logs")
}
