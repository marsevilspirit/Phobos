package log

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Level 定义日志级别
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
)

// String 返回日志级别的字符串表示
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	case PanicLevel:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel 解析日志级别字符串
func ParseLevel(s string) Level {
	switch s {
	case "debug", "DEBUG":
		return DebugLevel
	case "info", "INFO":
		return InfoLevel
	case "warn", "WARN":
		return WarnLevel
	case "error", "ERROR":
		return ErrorLevel
	case "fatal", "FATAL":
		return FatalLevel
	case "panic", "PANIC":
		return PanicLevel
	default:
		return InfoLevel
	}
}

// Fields 日志字段集合
type Fields map[string]interface{}

// Logger 增强的日志接口
type Logger interface {
	Debug(v ...any)
	Debugf(format string, v ...any)

	Info(v ...any)
	Infof(format string, v ...any)

	Warn(v ...any)
	Warnf(format string, v ...any)

	Error(v ...any)
	Errorf(format string, v ...any)

	Fatal(v ...any)
	Fatalf(format string, v ...any)

	Panic(v ...any)
	Panicf(format string, v ...any)

	// 新增结构化日志方法
	WithField(key string, value interface{}) Logger
	WithFields(fields Fields) Logger
	WithError(err error) Logger
	WithContext(ctx context.Context) Logger

	// 日志级别控制
	SetLevel(level Level)
	IsLevelEnabled(level Level) bool
}

// zapLogger Zap日志实现
type zapLogger struct {
	*zap.SugaredLogger
	level Level
}

// colorLevelEncoder 彩色级别编码器
func colorLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var color string
	switch l {
	case zapcore.DebugLevel:
		color = "\033[36m" // 青色
	case zapcore.InfoLevel:
		color = "\033[32m" // 绿色
	case zapcore.WarnLevel:
		color = "\033[33m" // 黄色
	case zapcore.ErrorLevel:
		color = "\033[31m" // 红色
	case zapcore.FatalLevel:
		color = "\033[35m" // 紫色
	case zapcore.PanicLevel:
		color = "\033[35m" // 紫色
	default:
		color = "\033[0m" // 默认
	}

	// 添加颜色前缀和重置后缀
	enc.AppendString(color + l.CapitalString() + "\033[0m")
}

// NewZapLogger 创建新的Zap日志器
func NewZapLogger(level Level) *zapLogger {
	// 确保日志目录存在
	os.MkdirAll("logs", 0755)

	// 创建控制台编码器配置（彩色）
	consoleEncoderConfig := zap.NewProductionEncoderConfig()
	consoleEncoderConfig.TimeKey = "timestamp"
	consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoderConfig.EncodeLevel = colorLevelEncoder
	consoleEncoderConfig.MessageKey = "message"
	consoleEncoderConfig.CallerKey = "caller"
	consoleEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	consoleEncoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	// 创建控制台编码器
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)

	// 创建文件编码器配置（JSON格式，无颜色）
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.TimeKey = "timestamp"
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEncoderConfig.MessageKey = "message"
	fileEncoderConfig.CallerKey = "caller"
	fileEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	fileEncoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	// 创建文件编码器
	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

	// 创建输出
	consoleOutput := zapcore.AddSync(os.Stdout)
	fileOutput, err := os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fileOutput = os.Stdout // 如果文件创建失败，使用标准输出
	}

	// 创建核心 - 同时输出到控制台和文件
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleOutput, zap.NewAtomicLevelAt(zapcore.Level(level))),
		zapcore.NewCore(fileEncoder, zapcore.AddSync(fileOutput), zap.NewAtomicLevelAt(zapcore.Level(level))),
	)

	// 创建日志器
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &zapLogger{
		SugaredLogger: logger.Sugar(),
		level:         level,
	}
}

// SetLevel 设置日志级别
func (l *zapLogger) SetLevel(level Level) {
	l.level = level
}

// IsLevelEnabled 检查日志级别是否启用
func (l *zapLogger) IsLevelEnabled(level Level) bool {
	return level >= l.level
}

// WithField 添加单个字段
func (l *zapLogger) WithField(key string, value interface{}) Logger {
	newLogger := &zapLogger{
		SugaredLogger: l.SugaredLogger.With(key, value),
		level:         l.level,
	}
	return newLogger
}

// WithFields 添加多个字段
func (l *zapLogger) WithFields(fields Fields) Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for key, value := range fields {
		args = append(args, key, value)
	}

	newLogger := &zapLogger{
		SugaredLogger: l.SugaredLogger.With(args...),
		level:         l.level,
	}
	return newLogger
}

// WithError 添加错误字段
func (l *zapLogger) WithError(err error) Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// WithContext 添加上下文字段
func (l *zapLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	// 提取请求ID等上下文信息
	if requestID := ctx.Value("request_id"); requestID != nil {
		return l.WithField("request_id", requestID)
	}

	return l
}

// 实现日志方法
func (l *zapLogger) Debug(v ...any) {
	if l.IsLevelEnabled(DebugLevel) {
		l.SugaredLogger.Debug(v...)
	}
}

func (l *zapLogger) Debugf(format string, v ...any) {
	if l.IsLevelEnabled(DebugLevel) {
		l.SugaredLogger.Debugf(format, v...)
	}
}

func (l *zapLogger) Info(v ...any) {
	if l.IsLevelEnabled(InfoLevel) {
		l.SugaredLogger.Info(v...)
	}
}

func (l *zapLogger) Infof(format string, v ...any) {
	if l.IsLevelEnabled(InfoLevel) {
		l.SugaredLogger.Infof(format, v...)
	}
}

func (l *zapLogger) Warn(v ...any) {
	if l.IsLevelEnabled(WarnLevel) {
		l.SugaredLogger.Warn(v...)
	}
}

func (l *zapLogger) Warnf(format string, v ...any) {
	if l.IsLevelEnabled(WarnLevel) {
		l.SugaredLogger.Warnf(format, v...)
	}
}

func (l *zapLogger) Error(v ...any) {
	if l.IsLevelEnabled(ErrorLevel) {
		l.SugaredLogger.Error(v...)
	}
}

func (l *zapLogger) Errorf(format string, v ...any) {
	if l.IsLevelEnabled(ErrorLevel) {
		l.SugaredLogger.Errorf(format, v...)
	}
}

func (l *zapLogger) Fatal(v ...any) {
	if l.IsLevelEnabled(FatalLevel) {
		l.SugaredLogger.Fatal(v...)
	}
}

func (l *zapLogger) Fatalf(format string, v ...any) {
	if l.IsLevelEnabled(FatalLevel) {
		l.SugaredLogger.Fatalf(format, v...)
	}
}

func (l *zapLogger) Panic(v ...any) {
	if l.IsLevelEnabled(PanicLevel) {
		l.SugaredLogger.Panic(v...)
	}
}

func (l *zapLogger) Panicf(format string, v ...any) {
	if l.IsLevelEnabled(PanicLevel) {
		l.SugaredLogger.Panicf(format, v...)
	}
}

// dummyLogger 虚拟日志实现
type dummyLogger struct{}

func (l *dummyLogger) Debug(v ...any)                                 {}
func (l *dummyLogger) Debugf(format string, v ...any)                 {}
func (l *dummyLogger) Info(v ...any)                                  {}
func (l *dummyLogger) Infof(format string, v ...any)                  {}
func (l *dummyLogger) Warn(v ...any)                                  {}
func (l *dummyLogger) Warnf(format string, v ...any)                  {}
func (l *dummyLogger) Error(v ...any)                                 {}
func (l *dummyLogger) Errorf(format string, v ...any)                 {}
func (l *dummyLogger) Fatal(v ...any)                                 {}
func (l *dummyLogger) Fatalf(format string, v ...any)                 {}
func (l *dummyLogger) Panic(v ...any)                                 {}
func (l *dummyLogger) Panicf(format string, v ...any)                 {}
func (l *dummyLogger) WithField(key string, value interface{}) Logger { return l }
func (l *dummyLogger) WithFields(fields Fields) Logger                { return l }
func (l *dummyLogger) WithError(err error) Logger                     { return l }
func (l *dummyLogger) WithContext(ctx context.Context) Logger         { return l }
func (l *dummyLogger) SetLevel(level Level)                           {}
func (l *dummyLogger) IsLevelEnabled(level Level) bool                { return false }

// 全局变量和函数
var l Logger = NewZapLogger(InfoLevel)

func SetLogger(logger Logger) {
	l = logger
}

func SetDummyLogger() {
	l = &dummyLogger{}
}

func SetLevel(level Level) {
	if logger, ok := l.(*zapLogger); ok {
		logger.SetLevel(level)
	}
}

func Debug(v ...any) {
	l.Debug(v...)
}

func Debugf(format string, v ...any) {
	l.Debugf(format, v...)
}

func Info(v ...any) {
	l.Info(v...)
}

func Infof(format string, v ...any) {
	l.Infof(format, v...)
}

func Warn(v ...any) {
	l.Warn(v...)
}

func Warnf(format string, v ...any) {
	l.Warnf(format, v...)
}

func Error(v ...any) {
	l.Error(v...)
}

func Errorf(format string, v ...any) {
	l.Errorf(format, v...)
}

func Fatal(v ...any) {
	l.Fatal(v...)
}

func Fatalf(format string, v ...any) {
	l.Fatalf(format, v...)
}

func Panic(v ...any) {
	l.Panic(v...)
}

func Panicf(format string, v ...any) {
	l.Panicf(format, v...)
}

// 结构化日志辅助函数
func WithField(key string, value interface{}) Logger {
	return l.WithField(key, value)
}

func WithFields(fields Fields) Logger {
	return l.WithFields(fields)
}

func WithError(err error) Logger {
	return l.WithError(err)
}

func WithContext(ctx context.Context) Logger {
	return l.WithContext(ctx)
}
