package server

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type TestService struct{}

type TestArg struct {
	Name string
}

type TestReply struct {
	Message string
}

func (s *TestService) TestMethod(ctx context.Context, arg *TestArg, reply *TestReply) error {
	if arg.Name == "" {
		return errors.New("name is empty")
	}
	reply.Message = "Hello, " + arg.Name
	return nil
}

func TestRegisterAndCall(t *testing.T) {
	server := &Server{}

	// 注册服务
	err := server.RegisterWithName("TestService", &TestService{}, "")
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// 获取注册的服务
	service := server.serviceMap["TestService"]
	if service == nil {
		t.Fatal("Service not found after registration")
	}

	// 获取注册的方法
	method := service.method["TestMethod"]
	if method == nil {
		t.Fatal("Method not found after registration")
	}

	// 准备参数和值
	ctx := context.Background()
	arg := TestArg{Name: "World"}
	argv := reflect.ValueOf(&arg)
	reply := TestReply{}
	replyv := reflect.ValueOf(&reply)

	// 调用方法
	err = service.call(ctx, method, argv, replyv)
	if err != nil {
		t.Fatalf("Method call failed: %v", err)
	}

	// 检查结果
	expectedMessage := "Hello, World"
	if reply.Message != expectedMessage {
		t.Fatalf("Expected message %v, but got %v", expectedMessage, reply.Message)
	}
}

func TestMethodWithEmptyName(t *testing.T) {
	server := &Server{}
	err := server.RegisterWithName("TestService", &TestService{}, "")
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	service := server.serviceMap["TestService"]
	method := service.method["TestMethod"]

	ctx := context.Background()
	arg := TestArg{Name: ""}
	argv := reflect.ValueOf(&arg)
	reply := TestReply{}
	replyv := reflect.ValueOf(&reply)

	err = service.call(ctx, method, argv, replyv)
	if err == nil || err.Error() != "name is empty" {
		t.Fatalf("Expected error 'name is empty', but got %v", err)
	}
}
