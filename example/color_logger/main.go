package main

import (
	"errors"
	"time"

	"github.com/marsevilspirit/phobos/log"
)

func main() {
	// 设置日志级别为调试
	log.SetLevel(log.DebugLevel)

	// 演示不同级别的彩色日志输出
	log.Info("=== 彩色日志演示 ===")

	// 调试级别 - 青色
	log.Debug("这是一条调试日志")
	log.Debugf("调试信息: 当前时间 %s", time.Now().Format("15:04:05"))

	// 信息级别 - 绿色
	log.Info("这是一条信息日志")
	log.Infof("信息: 程序启动成功，版本 %s", "1.0.0")

	// 警告级别 - 黄色
	log.Warn("这是一条警告日志")
	log.Warnf("警告: 配置文件 %s 不存在，使用默认配置", "config.yaml")

	// 错误级别 - 红色
	log.Error("这是一条错误日志")
	log.Errorf("错误: 连接数据库失败，错误信息: %s", "connection timeout")

	// 结构化日志
	log.WithField("service", "user-service").Info("用户服务启动")

	log.WithFields(log.Fields{
		"user_id":   "12345",
		"action":    "login",
		"ip":        "192.168.1.100",
		"timestamp": time.Now().Unix(),
	}).Info("用户登录成功")

	// 错误日志
	log.WithError(errors.New("网络连接失败")).Error("请求处理失败")

	// 演示不同颜色的日志级别
	log.Info("=== 颜色对比演示 ===")
	log.Debug("DEBUG - 青色")
	log.Info("INFO - 绿色")
	log.Warn("WARN - 黄色")
	log.Error("ERROR - 红色")

	// 测试所有日志级别
	log.Info("测试所有日志级别:")
	log.Debug("DEBUG级别日志")
	log.Info("INFO级别日志")
	log.Warn("WARN级别日志")
	log.Error("ERROR级别日志")

	log.Info("彩色日志演示完成！")
}
