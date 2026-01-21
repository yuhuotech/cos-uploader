package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Logger 简单的日志记录器
type Logger struct {
	stdout io.Writer
	file   io.Writer
}

// NewLogger 创建新的日志记录器（使用默认路径）
func NewLogger() *Logger {
	return NewLoggerWithPath("logs/cos-uploader.log")
}

// NewLoggerWithPath 创建新的日志记录器，同时输出到stdout和指定的文件路径
func NewLoggerWithPath(logPath string) *Logger {
	// 如果路径为空，使用默认路径
	if logPath == "" {
		logPath = "logs/cos-uploader.log"
	}

	// 创建日志目录
	logDir := filepath.Dir(logPath)
	os.MkdirAll(logDir, 0755)

	// 打开日志文件
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}

	return &Logger{
		stdout: os.Stdout,
		file:   logFile,
	}
}

// SetWriter 设置日志输出的写入器（用于测试）
func (l *Logger) SetWriter(stdout, file io.Writer) {
	l.stdout = stdout
	l.file = file
}

// formatLogMessage 格式化日志消息
// 格式: TIME [LEVEL] MESSAGE - key1=value1 key2=value2
func (l *Logger) formatLogMessage(level string, msg string, keysAndValues ...interface{}) string {
	// 构建时间和级别
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	levelStr := strings.ToUpper(level)
	logLine := fmt.Sprintf("%s [%s] %s", timeStr, levelStr, msg)

	// 添加键值对
	if len(keysAndValues) > 0 {
		logLine += " -"
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				key := keysAndValues[i]
				value := keysAndValues[i+1]
				logLine += fmt.Sprintf(" %v=%v", key, value)
			}
		}
	}

	return logLine + "\n"
}

// writeLog 写日志到stdout和文件
func (l *Logger) writeLog(message string) {
	l.stdout.Write([]byte(message))
	l.file.Write([]byte(message))
}

// Info 记录info级别日志
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	logMsg := l.formatLogMessage("info", msg, keysAndValues...)
	l.writeLog(logMsg)
}

// Error 记录error级别日志
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	logMsg := l.formatLogMessage("error", msg, keysAndValues...)
	l.writeLog(logMsg)
}

// Warn 记录warn级别日志
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	logMsg := l.formatLogMessage("warn", msg, keysAndValues...)
	l.writeLog(logMsg)
}

// Debug 记录debug级别日志
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	logMsg := l.formatLogMessage("debug", msg, keysAndValues...)
	l.writeLog(logMsg)
}

// Sync 同步日志（关闭文件）
func (l *Logger) Sync() error {
	if f, ok := l.file.(*os.File); ok {
		return f.Close()
	}
	return nil
}
