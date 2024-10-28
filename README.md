# m_RPC

##### 2024-10-28 bug
当客户端断开连接时, server报error
```
2024/10/28 18:27:02 server.go:203: ERROR: mrpc: failed to read request: EOF
```
其实也不算bug,只是一时这样处理
