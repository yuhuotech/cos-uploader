package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 封装的日志记录器
type Logger struct {
	*zap.SugaredLogger
}

// NewLogger 创建新的日志记录器，同时输出到stdout和文件
func NewLogger() *Logger {
	// 创建日志目录
	os.MkdirAll("logs", 0755)

	// 配置encoder
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建stdout core
	stdoutWriter := zapcore.AddSync(os.Stdout)
	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		stdoutWriter,
		zapcore.InfoLevel,
	)

	// 创建文件core
	logFile, _ := os.OpenFile("logs/cos-uploader.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(logFile),
		zapcore.DebugLevel,
	)

	// 合并cores
	core := zapcore.NewTee(stdoutCore, fileCore)
	logger := zap.New(core, zap.AddCaller())

	return &Logger{logger.Sugar()}
}

// Info 记录info级别日志
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.SugaredLogger.Infow(msg, keysAndValues...)
}

// Error 记录error级别日志
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.SugaredLogger.Errorw(msg, keysAndValues...)
}

// Warn 记录warn级别日志
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.SugaredLogger.Warnw(msg, keysAndValues...)
}

// Debug 记录debug级别日志
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.SugaredLogger.Debugw(msg, keysAndValues...)
}
