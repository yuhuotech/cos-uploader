package watcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hmw/cos-uploader/logger"
	"go.uber.org/zap"
)

func TestNewWatcher(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建一个mock logger
	zapLogger := zap.NewNop()
	log := &logger.Logger{zapLogger.Sugar()}
	watcher, err := NewWatcher([]string{tmpDir}, []string{"create", "write"}, log)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	if watcher == nil {
		t.Fatal("Expected watcher, got nil")
	}
}

func TestShouldWatch(t *testing.T) {
	tests := []struct {
		name       string
		events     []string
		checkEvent string
		expected   bool
	}{
		{
			name:       "should watch create",
			events:     []string{"create", "write"},
			checkEvent: "create",
			expected:   true,
		},
		{
			name:       "should not watch remove",
			events:     []string{"create", "write"},
			checkEvent: "remove",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Watcher{events: tt.events}
			result := w.shouldWatch(tt.checkEvent)
			if result != tt.expected {
				t.Errorf("shouldWatch(%s) = %v, expected %v", tt.checkEvent, result, tt.expected)
			}
		})
	}
}

func TestIsInWatchedDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建一个mock logger
	zapLogger := zap.NewNop()
	log := &logger.Logger{zapLogger.Sugar()}

	watcher, err := NewWatcher([]string{tmpDir}, []string{"create"}, log)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// 测试监听目录内的文件
	testFile := filepath.Join(tmpDir, "test.txt")
	if !watcher.IsInWatchedDirectory(testFile) {
		t.Errorf("Expected file in watched directory, but IsInWatchedDirectory returned false")
	}

	// 测试不在监听目录内的文件
	otherFile := filepath.Join(os.TempDir(), "other.txt")
	if watcher.IsInWatchedDirectory(otherFile) {
		t.Errorf("Expected file not in watched directory, but IsInWatchedDirectory returned true")
	}
}
