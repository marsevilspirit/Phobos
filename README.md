### m_RPC

通过对net/rpc的源码学习，进行扩展开发

##### 特点

1.能进行基本的RPC调用  
2.支持多种序列化协议  
3.使用Gzip对过长的data进行压缩  
4.支持http与rpc协议之间转换  
5.支持超时处理  
6.使用较为灵活的元数据(metadata)传递data  
7.支持etcd做服务注册和服务发现  
8.支持多种负载均衡  
9.支持熔断

##### 使用方法

rpc定义:
```go
type Args struct {
	First string
}

type Reply struct {
	Second string
}

type HelloWorld int

func (t *HelloWorld) Greet(ctx context.Context, args *Args, reply *Reply) error {
	reply.Second = "Hello " + args.First
	return nil
}
```

server:
```go
func main() {
	s := server.NewServer(nil)
	s.RegisterWithName("HelloWorld", new(example.HelloWorld), "")
	s.Serve("tcp", *addr)
}
```

client:
```go
func main() {
	d := client.NewP2PDiscovery("tcp@"+*addr, "")
	xclient := client.NewXClient("HelloWorld", "Greet", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer xclient.Close()

	args := &example.Args{
		First: "budei",
	}

	reply := &example.Reply{}

	err := xclient.Call(context.Background(), args, reply, nil)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	log.Print("reply: ", reply.Second)
}
```
