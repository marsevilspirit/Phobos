# 彩色日志系统

这是一个基于Zap的彩色日志系统，支持在终端中显示不同颜色的日志级别，同时将日志保存到文件中。

## 功能特性

- 🎨 **彩色输出**: 不同日志级别使用不同颜色
- 📝 **双重输出**: 控制台彩色输出 + 文件JSON格式保存
- 🔧 **灵活配置**: 支持自定义日志级别和格式
- 📊 **结构化日志**: 支持字段和上下文信息
- 🚀 **高性能**: 基于Zap高性能日志库

## 颜色方案

| 日志级别 | 颜色 | 颜色代码 | 说明 |
|---------|------|----------|------|
| DEBUG   | 青色 | `\033[36m` | 调试信息 |
| INFO    | 绿色 | `\033[32m` | 一般信息 |
| WARN    | 黄色 | `\033[33m` | 警告信息 |
| ERROR   | 红色 | `\033[31m` | 错误信息 |
| FATAL   | 紫色 | `\033[35m` | 致命错误 |
| PANIC   | 紫色 | `\033[35m` | 程序崩溃 |

## 使用方法

### 基本用法

```go
package main

import "github.com/marsevilspirit/phobos/log"

func main() {
    // 设置日志级别
    log.SetLevel(log.DebugLevel)
    
    // 输出不同级别的日志
    log.Debug("调试信息")
    log.Info("一般信息")
    log.Warn("警告信息")
    log.Error("错误信息")
}
```

### 格式化日志

```go
log.Infof("用户 %s 登录成功，IP: %s", username, ip)
log.Errorf("连接失败: %v", err)
```

### 结构化日志

```go
// 添加单个字段
log.WithField("user_id", "123").Info("用户操作")

// 添加多个字段
log.WithFields(log.Fields{
    "service": "auth",
    "method":  "login",
    "ip":      "192.168.1.100",
}).Info("认证请求")

// 添加错误信息
log.WithError(err).Error("操作失败")
```

### 上下文日志

```go
ctx := context.WithValue(context.Background(), "request_id", "req-123")
log.WithContext(ctx).Info("处理请求")
```

## 配置选项

### 日志级别

```go
// 可用的日志级别
log.DebugLevel  // 调试
log.InfoLevel   // 信息（默认）
log.WarnLevel   // 警告
log.ErrorLevel  // 错误
log.FatalLevel  // 致命
log.PanicLevel  // 崩溃
```

### 自定义日志器

```go
// 创建自定义日志器
logger := log.NewZapLogger(log.DebugLevel)

// 设置全局日志器
log.SetLogger(logger)

// 使用虚拟日志器（不输出）
log.SetDummyLogger()
```

## 输出格式

### 控制台输出（彩色）

```
2025-08-12T23:22:33+0800    INFO    用户登录成功    {"user_id": "12345", "action": "login"}
```

### 文件输出（JSON格式）

```json
{
  "level": "info",
  "timestamp": "2025-08-12T23:22:33+0800",
  "caller": "main.go:25",
  "message": "用户登录成功",
  "user_id": "12345",
  "action": "login"
}
```

## 文件输出

日志文件默认保存在 `logs/` 目录下：
- `logs/app.log` - 应用程序日志
- 如果目录不存在，系统会自动创建

## 性能特性

- 异步日志写入
- 内存池复用
- 结构化字段缓存
- 最小化内存分配

## 注意事项

1. **颜色支持**: 彩色输出仅在支持ANSI颜色代码的终端中显示
2. **文件权限**: 确保程序有权限创建和写入logs目录
3. **性能影响**: 彩色编码器对性能影响很小，但在高并发场景下建议使用文件日志
4. **日志轮转**: 当前版本不包含日志轮转功能，建议配合外部工具使用

## 示例程序

运行示例程序查看彩色日志效果：

```bash
go run example/color_logger/main.go
```

## 依赖

- `go.uber.org/zap` - 高性能日志库
- `go.uber.org/zap/zapcore` - Zap核心功能
