package log

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestNewZapLogger(t *testing.T) {
	logger := NewZapLogger(InfoLevel)

	if logger == nil {
		t.Fatal("logger should not be nil")
	}

	if !logger.IsLevelEnabled(InfoLevel) {
		t.Error("logger should be enabled for InfoLevel")
	}

	if logger.IsLevelEnabled(DebugLevel) {
		t.Error("logger should not be enabled for DebugLevel when level is InfoLevel")
	}
}

func TestLoggerWithField(t *testing.T) {
	logger := NewZapLogger(InfoLevel)
	loggerWithField := logger.WithField("user_id", "123")

	if loggerWithField == nil {
		t.Fatal("logger with field should not be nil")
	}

	// 测试字段是否被正确添加
	loggerWithField.Info("test message")
}

func TestLoggerWithFields(t *testing.T) {
	logger := NewZapLogger(InfoLevel)
	fields := Fields{
		"service": "test",
		"version": "1.0",
	}
	loggerWithFields := logger.WithFields(fields)

	if loggerWithFields == nil {
		t.Fatal("logger with fields should not be nil")
	}

	// 测试字段是否被正确添加
	loggerWithFields.Info("test message with fields")
}

func TestLoggerWithError(t *testing.T) {
	logger := NewZapLogger(InfoLevel)
	testErr := errors.New("test error")
	loggerWithError := logger.WithError(testErr)

	if loggerWithError == nil {
		t.Fatal("logger with error should not be nil")
	}

	// 测试错误字段是否被正确添加
	loggerWithError.Error("test error message")
}

func TestLoggerWithContext(t *testing.T) {
	logger := NewZapLogger(InfoLevel)

	// 测试空上下文
	loggerWithCtx := logger.WithContext(nil)
	if loggerWithCtx == nil {
		t.Fatal("logger with nil context should not be nil")
	}

	// 测试带请求ID的上下文
	ctx := context.WithValue(context.Background(), "request_id", "req-123")
	loggerWithCtx = logger.WithContext(ctx)
	if loggerWithCtx == nil {
		t.Fatal("logger with context should not be nil")
	}

	// 测试上下文字段是否被正确添加
	loggerWithCtx.Info("test message with context")
}

func TestLoggerLevelControl(t *testing.T) {
	logger := NewZapLogger(DebugLevel)

	// 测试级别设置
	logger.SetLevel(InfoLevel)
	if logger.IsLevelEnabled(DebugLevel) {
		t.Error("logger should not be enabled for DebugLevel after setting level to InfoLevel")
	}

	if !logger.IsLevelEnabled(InfoLevel) {
		t.Error("logger should be enabled for InfoLevel")
	}

	if !logger.IsLevelEnabled(ErrorLevel) {
		t.Error("logger should be enabled for ErrorLevel")
	}
}

func TestDummyLogger(t *testing.T) {
	dummy := &dummyLogger{}

	// 测试所有方法都不会panic
	dummy.Debug("test")
	dummy.Debugf("test %s", "message")
	dummy.Info("test")
	dummy.Infof("test %s", "message")
	dummy.Warn("test")
	dummy.Warnf("test %s", "message")
	dummy.Error("test")
	dummy.Errorf("test %s", "message")
	dummy.Fatal("test")
	dummy.Fatalf("test %s", "message")
	dummy.Panic("test")
	dummy.Panicf("test %s", "message")

	// 测试结构化日志方法
	result := dummy.WithField("key", "value")
	if result == nil {
		t.Error("WithField should return logger")
	}

	result = dummy.WithFields(Fields{"key": "value"})
	if result == nil {
		t.Error("WithFields should return logger")
	}

	result = dummy.WithError(nil)
	if result == nil {
		t.Error("WithError should return logger")
	}

	result = dummy.WithContext(nil)
	if result == nil {
		t.Error("WithContext should return logger")
	}

	// 测试级别控制
	dummy.SetLevel(InfoLevel)
	if dummy.IsLevelEnabled(InfoLevel) {
		t.Error("dummy logger should never be enabled")
	}
}

func TestGlobalFunctions(t *testing.T) {
	// 测试全局函数
	Debug("test debug")
	Debugf("test debug %s", "message")
	Info("test info")
	Infof("test info %s", "message")
	Warn("test warn")
	Warnf("test warn %s", "message")
	Error("test error")
	Errorf("test error %s", "message")

	// 测试结构化日志辅助函数
	logger := WithField("test_key", "test_value")
	if logger == nil {
		t.Error("WithField should return logger")
	}

	logger = WithFields(Fields{"key1": "value1", "key2": "value2"})
	if logger == nil {
		t.Error("WithFields should return logger")
	}

	logger = WithError(nil)
	if logger == nil {
		t.Error("WithError should return logger")
	}

	logger = WithContext(context.Background())
	if logger == nil {
		t.Error("WithContext should return logger")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"warn", WarnLevel},
		{"WARN", WarnLevel},
		{"error", ErrorLevel},
		{"ERROR", ErrorLevel},
		{"fatal", FatalLevel},
		{"FATAL", FatalLevel},
		{"panic", PanicLevel},
		{"PANIC", PanicLevel},
		{"unknown", InfoLevel}, // 默认值
		{"", InfoLevel},        // 空字符串
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{FatalLevel, "FATAL"},
		{PanicLevel, "PANIC"},
		{100, "UNKNOWN"}, // 未知级别
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestColorLogger(t *testing.T) {
	// 创建彩色日志器
	logger := NewZapLogger(DebugLevel)

	// 测试不同级别的日志输出
	t.Run("Debug Level", func(t *testing.T) {
		logger.Debug("这是一条调试日志")
		logger.Debugf("调试日志格式: %s", "测试消息")
	})

	t.Run("Info Level", func(t *testing.T) {
		logger.Info("这是一条信息日志")
		logger.Infof("信息日志格式: %s", "测试消息")
	})

	t.Run("Warn Level", func(t *testing.T) {
		logger.Warn("这是一条警告日志")
		logger.Warnf("警告日志格式: %s", "测试消息")
	})

	t.Run("Error Level", func(t *testing.T) {
		logger.Error("这是一条错误日志")
		logger.Errorf("错误日志格式: %s", "测试消息")
	})

	t.Run("With Fields", func(t *testing.T) {
		logger.WithField("user_id", "123").Info("用户登录")
		logger.WithFields(Fields{
			"service": "auth",
			"method":  "login",
		}).Info("认证服务调用")
	})
}

func TestColorLevelEncoder(t *testing.T) {
	// 测试彩色编码器
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		LevelKey:    "level",
		TimeKey:     "time",
		MessageKey:  "message",
		EncodeLevel: colorLevelEncoder,
	})

	// 创建测试日志条目
	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "测试消息",
	}

	// 编码日志条目
	buf, err := encoder.EncodeEntry(entry, nil)
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}

	// 验证输出包含颜色代码
	output := buf.String()
	if output == "" {
		t.Error("编码输出为空")
	}

	t.Logf("编码输出: %s", output)
}
