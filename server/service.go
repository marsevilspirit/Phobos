package server

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/marsevilspirit/m_RPC/log"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

var typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()

type methodType struct {
	sync.Mutex
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
	// numCalls  uint
}

type functionType struct {
	sync.Mutex
	fn        reflect.Value
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name     string
	rcvr     reflect.Value          // receiver of methods for the service
	typ      reflect.Type           // type of the receiver
	method   map[string]*methodType // 注册方法
	function map[string]*functionType
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	// 解引用指针类型，直到得到非指针类型
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// t.PkgPath() == "" 的判断是用来检查类型是否为内建类型或未命名类型
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func (s *Server) Register(rcvr interface{}, metadata string) error {
	s.Plugins.DoRegister("", rcvr, metadata)
	return s.register(rcvr, "", false)
}

func (s *Server) RegisterWithName(name string, rcvr interface{}, metadata string) error {
	if s.Plugins == nil {
		s.Plugins = &pluginContainer{}
	}
	s.Plugins.DoRegister(name, rcvr, metadata)
	return s.register(rcvr, name, true)
}

func (s *Server) RegisterFunction(servicePath string, fn interface{}, metadata string) error {
	s.Plugins.DoRegisterFunction("", fn, metadata)
	return s.registerFunction(servicePath, "", fn, false)
}

func (s *Server) RegisterFunctionWithName(servicePath string, name string, fn interface{}, metadata string) error {
	if s.Plugins == nil {
		s.Plugins = &pluginContainer{}
	}
	s.Plugins.DoRegisterFunction(name, fn, metadata)
	return s.registerFunction(servicePath, name, fn, true)
}

func (s *Server) register(rcvr interface{}, name string, useName bool) error {
	s.serviceMapMu.Lock()
	defer s.serviceMapMu.Unlock()

	if s.serviceMap == nil {
		s.serviceMap = make(map[string]*service)
	}

	service := new(service)
	service.typ = reflect.TypeOf(rcvr)
	service.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(service.rcvr).Type().Name()

	if useName {
		sname = name
	}

	if sname == "" {
		errorStr := "mrpc.register: no service name for type " + service.typ.String()
		log.Error(errorStr)
		return errors.New(errorStr)
	}

	if !useName && !ast.IsExported(sname) {
		errorStr := "mrpc.register: type " + sname + " is not exported"
		log.Error(errorStr)
		return errors.New(errorStr)
	}

	service.name = sname

	service.method = suitableMethods(service.typ, true)

	if len(service.method) == 0 {
		var errorStr string

		method := suitableMethods(reflect.PointerTo(service.typ), false)
		if len(method) != 0 {
			errorStr = "mrpc.register: type " + sname + " has no exportedmethods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			errorStr = "mrpc.register: type " + sname + " has no exportedmethods of suitable type"
		}
		log.Error(errorStr)
		return errors.New(errorStr)
	}
	s.serviceMap[service.name] = service
	return nil
}

func (s *Server) registerFunction(servicePath string, name string, fn interface{}, useName bool) error {
	s.serviceMapMu.Lock()
	defer s.serviceMapMu.Unlock()
	if s.serviceMap == nil {
		s.serviceMap = make(map[string]*service)
	}

	ss := s.serviceMap[servicePath]
	if ss == nil {
		ss = new(service)
		ss.name = servicePath
		ss.function = make(map[string]*functionType)
	}

	f, ok := fn.(reflect.Value)
	if !ok {
		f = reflect.ValueOf(fn)
	}
	if f.Kind() != reflect.Func {
		return errors.New("mrpc.registerFunction: not a function")
	}

	fname := runtime.FuncForPC(reflect.Indirect(f).Pointer()).Name()
	if fname == "" {
		i := strings.LastIndex(fname, ".")
		if i >= 0 {
			fname = fname[i+1:]
		}
	}

	if useName {
		fname = name
	}
	if fname == "" {
		error := "mrpc.registerFunction: no function name for type " + f.String()
		log.Error(error)
		return errors.New(error)
	}

	t := f.Type()
	if t.NumIn() != 3 {
		return fmt.Errorf("mrpc.registerFunction: has wrong number of ins: %s", f.Type().String())
	}
	if t.NumOut() != 1 {
		return fmt.Errorf("mrpc.registerFunction: has wrong number of outs: %s", f.Type().String())
	}

	ctxType := t.In(0)
	if !ctxType.Implements(typeOfContext) {
		return fmt.Errorf("mrpc.registerFunction: first argument not of type context.Context")
	}

	argType := t.In(1)
	if !isExportedOrBuiltinType(argType) {
		return fmt.Errorf("mrpc.registerFunction: argument type not exported: %s", argType)
	}

	replyType := t.In(2)
	if replyType.Kind() != reflect.Ptr {
		return fmt.Errorf("mrpc.registerFunction: reply type not a pointer: %s", replyType)
	}
	if !isExportedOrBuiltinType(replyType) {
		return fmt.Errorf("mrpc.registerFunction: reply type not exported: %s", replyType)
	}

	if returnType := t.Out(0); returnType != typeOfError {
		return fmt.Errorf("mrpc.registerFunction: returns %s not error", returnType.String())
	}

	ss.function[fname] = &functionType{fn: f, ArgType: argType, ReplyType: replyType}

	s.serviceMap[""] = ss
	return nil
}

func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)

	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name

		if method.PkgPath != "" {
			continue
		}

		// receiver, context.Context, *args, *reply
		if mtype.NumIn() != 4 {
			if reportErr {
				log.Info("method", mname, "has wrong number of ins:", mtype.NumIn())
			}
			continue
		}

		ctxType := mtype.In(1)
		if !ctxType.Implements(typeOfContext) {
			if reportErr {
				log.Info("method", mname, " must use context.Context as the first parameter")
			}
			continue
		}

		argType := mtype.In(2)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				log.Info(mname, "parameter type not exported:", argType)
			}
			continue
		}

		// must be a pointer
		replyType := mtype.In(3)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				log.Info("method", mname, "reply type not a pointer")
			}
			continue
		}

		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				log.Info("method", mname, "reply type not exported:", replyType)
			}
			continue
		}

		if mtype.NumOut() != 1 {
			if reportErr {
				log.Info("method", mname, "has wrong number of outs: ", mtype.NumOut())
			}
			continue
		}

		if returnType := mtype.Out(0); returnType != typeOfError {
			if reportErr {
				log.Info("method", mname, "returns", returnType.String(), "not error")
			}
			continue
		}

		methods[mname] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
	}

	return methods
}

func (s *service) call(ctx context.Context, mtype *methodType, argv, replyv reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[service internal] error: %v", r)
		}
	}()

	f := mtype.method.Func

	returnValues := f.Call([]reflect.Value{s.rcvr, reflect.ValueOf(ctx), argv, replyv})

	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}

	return nil
}

func (s *service) callForFunction(ctx context.Context, ft *functionType, argv, replyv reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[service internal] error: %v", r)
		}
	}()

	returnValues := ft.fn.Call([]reflect.Value{reflect.ValueOf(ctx), argv, replyv})

	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}

	return nil
}
