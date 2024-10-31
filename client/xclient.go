package client

import (
	"context"
	"errors"
	"strings"
	"sync"

	ex "github.com/marsevilspirit/m_RPC/errors"
)

var (
	ErrXClientShutdown = errors.New("xClient is shut down")
)

type XClient interface {
	Go(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, done chan *Call) (*Call, error)
	Call(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error
	Close() error
}

type KVPair struct {
	Key   string
	Value string
}

// ServiceDiscovery 接口定义，用于服务发现
type ServiceDiscovery interface {
	// 获取所有服务
	GetServices() []*KVPair
	// 监听服务变化，返回服务变化的通道
	WatchService() chan []*KVPair
}

// xClient 结构体实现 XClient 接口
type xClient struct {
	failMode     FailMode           // 失败处理模式
	selectMode   SelectMode         // 选择处理模式
	cachedClient map[string]*Client // 缓存的客户端连接

	mu        sync.RWMutex      // 读写锁，用于保护共享资源的并发访问
	servers   map[string]string // 当前已知的服务器地址
	discovery ServiceDiscovery  // 服务发现接口

	isShutdown bool // 客户端是否已关闭的标志
}

// NewXClient 工厂函数，用于创建 xClient 实例
func NewXClient(failMode FailMode, selectMode SelectMode, discovery ServiceDiscovery) XClient {
	// 初始化 xClient 结构体
	client := &xClient{
		failMode:   failMode,
		selectMode: selectMode,
		discovery:  discovery,
	}

	// 启动一个 Goroutine 来监控服务的变化
	go client.watch()

	// 更新服务列表
	servers := make(map[string]string)
	pairs := discovery.GetServices()
	for _, p := range pairs {
		servers[p.Key] = p.Value
	}
	client.servers = servers

	return client
}

// watch 方法，用于不断监听服务变化并更新服务器列表
func (c *xClient) watch() {
	ch := c.discovery.WatchService()
	for pairs := range ch {
		servers := make(map[string]string)
		for _, p := range pairs {
			servers[p.Key] = p.Value
		}
		c.mu.Lock()
		c.servers = servers
		c.mu.Unlock()
	}
}

// selectClient 方法，用于根据选择模式选择客户端
func (c *xClient) selectClient() (*Client, error) {
	// TODO: 根据选择模式获取服务器键
	k := ""

	return c.getCachedClient(k)
}

// getCachedClient 方法，根据服务器键获取缓存的客户端连接
func (c *xClient) getCachedClient(k string) (*Client, error) {
	c.mu.RLock()
	client := c.cachedClient[k]
	if client != nil {
		if !client.closing && !client.shutdown {
			c.mu.RUnlock()
			return client, nil
		}
	}

	// 双检查，确保线程安全
	c.mu.Lock()
	client = c.cachedClient[k]
	if client == nil {
		network, addr := splitNetworkAndAddress(k)
		client = &Client{
			// TODO: 初始化这个客户端
		}
		err := client.Connect(network, addr)
		if err != nil {
			c.mu.Unlock()
			return nil, err
		}
		c.cachedClient[k] = client
	}
	c.mu.Unlock()

	return client, nil
}

// splitNetworkAndAddress 方法，用于分割服务器地址
func splitNetworkAndAddress(server string) (string, string) {
	ss := strings.SplitN(server, "@", 2)
	if len(ss) == 1 {
		return "tcp", server
	}

	return ss[0], ss[1]
}

// Go 方法实现异步调用 RPC
func (c *xClient) Go(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, done chan *Call) (*Call, error) {
	if c.isShutdown {
		return nil, ErrXClientShutdown
	}
	client, err := c.selectClient()
	if err != nil {
		return nil, err
	}
	return client.Go(ctx, servicePath, serviceMethod, args, reply, done), nil
}

// Call 方法实现同步调用 RPC，通过调用 Go 方法并等待结果
func (c *xClient) Call(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error {
	if c.isShutdown {
		return ErrXClientShutdown
	}

	client, err := c.selectClient()
	if err != nil {
		return err
	}

	return client.Call(ctx, servicePath, serviceMethod, args, reply)
}

// Close 方法关闭客户端，释放资源
func (c *xClient) Close() error {
	c.isShutdown = true

	var errs []error
	c.mu.Lock()
	for _, v := range c.cachedClient {
		e := v.Close()
		if e != nil {
			errs = append(errs, e)
		}
	}
	c.mu.Unlock()

	if len(errs) > 0 {
		return ex.NewMultiError(errs)
	}
	return nil
}

