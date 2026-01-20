package watcher

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hmw/cos-uploader/logger"
)

// Event 文件变更事件
type Event struct {
	FilePath string // 文件绝对路径
	Type     string // 事件类型: create, write, remove, rename, chmod
	Time     int64  // 事件时间戳（纳秒）
}

// Watcher 文件监听器
type Watcher struct {
	watcher     *fsnotify.Watcher
	directories []string
	events      []string // 监听的事件类型
	eventsChan  chan Event
	logger      *logger.Logger
	done        chan struct{}
}

// NewWatcher 创建新的文件监听器
func NewWatcher(directories []string, watchEvents []string, log *logger.Logger) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		watcher:     fsWatcher,
		directories: directories,
		events:      watchEvents,
		eventsChan:  make(chan Event, 100),
		logger:      log,
		done:        make(chan struct{}),
	}

	// 递归添加所有目录
	for _, dir := range directories {
		if err := w.addRecursive(dir); err != nil {
			fsWatcher.Close()
			return nil, err
		}
	}

	return w, nil
}

// addRecursive 递归添加目录和子目录
func (w *Watcher) addRecursive(dir string) error {
	// 这里简化处理，实际应该遍历所有子目录
	// 为了演示，先添加顶级目录
	if err := w.watcher.Add(dir); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", dir, err)
	}
	w.logger.Debug("Watching directory", "path", dir)
	return nil
}

// Start 启动监听
func (w *Watcher) Start() {
	go func() {
		for {
			select {
			case fsEvent, ok := <-w.watcher.Events:
				if !ok {
					return
				}

				eventType := w.getEventType(fsEvent)
				if !w.shouldWatch(eventType) {
					continue
				}

				event := Event{
					FilePath: fsEvent.Name,
					Type:     eventType,
					Time:     time.Now().UnixNano(),
				}

				w.logger.Debug("File event detected", "file", fsEvent.Name, "type", eventType)
				select {
				case w.eventsChan <- event:
				case <-w.done:
					return
				}

			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				w.logger.Error("Watcher error", "error", err)
			}
		}
	}()
}

// getEventType 获取事件类型名称
func (w *Watcher) getEventType(fsEvent fsnotify.Event) string {
	switch {
	case fsEvent.Op&fsnotify.Create == fsnotify.Create:
		return "create"
	case fsEvent.Op&fsnotify.Write == fsnotify.Write:
		return "write"
	case fsEvent.Op&fsnotify.Remove == fsnotify.Remove:
		return "remove"
	case fsEvent.Op&fsnotify.Rename == fsnotify.Rename:
		return "rename"
	case fsEvent.Op&fsnotify.Chmod == fsnotify.Chmod:
		return "chmod"
	default:
		return "unknown"
	}
}

// shouldWatch 判断是否应该监听该事件
func (w *Watcher) shouldWatch(eventType string) bool {
	for _, e := range w.events {
		if e == eventType {
			return true
		}
	}
	return false
}

// Events 返回事件通道
func (w *Watcher) Events() <-chan Event {
	return w.eventsChan
}

// Close 关闭监听器
func (w *Watcher) Close() error {
	close(w.done)
	close(w.eventsChan)
	return w.watcher.Close()
}

// IsInWatchedDirectory 检查文件是否在监听的目录中
func (w *Watcher) IsInWatchedDirectory(filePath string) bool {
	absPath, _ := filepath.Abs(filePath)
	for _, dir := range w.directories {
		absDir, _ := filepath.Abs(dir)
		if strings.HasPrefix(absPath, absDir) {
			return true
		}
	}
	return false
}
