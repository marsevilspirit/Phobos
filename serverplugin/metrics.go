package serverplugin

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/marsevilspirit/m_RPC/protocol"
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsPlugin 实现了多个插件接口
type MetricsPlugin struct {
	mu              sync.Mutex
	acceptedConns   prometheus.Counter
	processedReqs   prometheus.Counter
	requestDuration prometheus.Histogram
}

// NewMetricsPlugin 创建一个新的 MetricsPlugin 实例
func NewMetricsPlugin() *MetricsPlugin {
	mp := &MetricsPlugin{
		// 接受的连接数
		acceptedConns: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "accepted_connections_total",
			Help: "Total number of accepted connections.",
		}),
		// 处理的请求数
		processedReqs: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "processed_requests_total",
			Help: "Total number of processed requests.",
		}),
		// 请求处理时间
		requestDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Histogram of request processing durations in seconds.",
			Buckets: prometheus.DefBuckets, // 使用默认桶
		}),
	}

	// 注册指标
	prometheus.MustRegister(mp.acceptedConns)
	prometheus.MustRegister(mp.processedReqs)
	prometheus.MustRegister(mp.requestDuration)

	return mp
}

// Register 实现 RegisterPlugin 接口
func (mp *MetricsPlugin) Register(name string, rcvr interface{}, metadata string) error {
	fmt.Printf("Plugin MetricsPlugin registered with metadata: {%s}\n", metadata)
	return nil
}

// HandleConnAccept 实现 PostConnAcceptPlugin 接口
func (mp *MetricsPlugin) HandleConnAccept(conn net.Conn) (net.Conn, bool) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.acceptedConns.Inc() // 增加连接计数
	return conn, true
}

// PreReadRequest 实现 PreReadRequestPlugin 接口
func (mp *MetricsPlugin) PreReadRequest(ctx context.Context) error {
	return nil
}

// PostReadRequest 实现 PostReadRequestPlugin 接口
func (mp *MetricsPlugin) PostReadRequest(ctx context.Context, r *protocol.Message, e error) error {
	if e != nil {
		fmt.Println("Error reading request:", e)
		return e
	}
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.processedReqs.Inc() // 增加请求计数
	return nil
}

// // PreWriteResponse 实现 PreWriteResponsePlugin 接口
// func (mp *MetricsPlugin) PreWriteResponse(ctx context.Context, msg *protocol.Message) error {
// 	fmt.Println("metrics Pre-writing response")
// 	return nil
// }
//
// // PostWriteResponse 实现 PostWriteResponsePlugin 接口
// func (mp *MetricsPlugin) PostWriteResponse(ctx context.Context, req *protocol.Message, resp *protocol.Message, e error) error {
//
// 	fmt.Println("metrics Post-writing response")
// 	return nil
// }
