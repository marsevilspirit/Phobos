### m_RPC

通过对net/rpc的源码学习，进行扩展开发

##### 特点

1.能进行基本的RPC调用  
2.支持多种序列化协议  
3.使用Gzip对过长的data进行压缩  
4.支持http与rpc协议之间转换  
5.支持超时处理  
6.使用较为灵活的元数据(metadata)传递data  
7.支持etcd做服务注册
8.支持多重负载均衡

##### 使用方法

rpc定义:
```go
type HelloWorld struct{}

type HelloWorldArgs struct {
	First string
}

type HelloWorldReply struct {
	Last string
}

func (h *HelloWorld) Helloworld(ctx context.Context, args *HelloWorldArgs, reply *HelloWorldReply) error {
	reply.Last = args.First + " world!"
	return nil
}

```

server:
```go
func main() {
	server := server.Server{}
	server.Register(new(helloworld.HelloWorld))
	server.Serve("tcp", "127.0.0.1:50000")
	defer server.Close()
}

```

client:
```go
func main() {
	client := client.Client{
		SerializeType: protocol.JSON,
		CompressType:  protocol.Gzip,
	}

	err := client.Connect("tcp", "127.0.0.1:50000")
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	args := &helloworld.HelloWorldArgs{
		First: "hello",
	}

	reply := &helloworld.HelloWorldReply{}

	err = client.Call(context.Background(), "HelloWorld", "Helloworld", args, reply)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	fmt.Println("reply:", reply.Last)
}

```

##### 2024-10-28 bug
1.当客户端断开连接时, server报error。
```
2024/10/28 18:27:02 server.go:203: ERROR: mrpc: failed to read request: EOF
```
其实也不算bug,只是一时这样处理。

2.因为gogoprotobuf被弃用，研究了半天为啥protobuf协议不通。
改成官方的后，成功修复。
