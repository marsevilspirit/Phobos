package server

import (
	"reflect"
	"sync"
)

var UsePool bool

type Reset interface {
	Reset()
}

var argsReplyPools = &typePools{
	pools: make(map[reflect.Type]*sync.Pool),
	New: func(t reflect.Type) any {
		var argv reflect.Value

		if t.Kind() == reflect.Ptr {
			argv = reflect.New(t.Elem())
		} else {
			argv = reflect.New(t)
		}

		return argv.Interface()
	},
}

type typePools struct {
	pools map[reflect.Type]*sync.Pool
	New   func(t reflect.Type) any
}

func (p *typePools) Init(t reflect.Type) {
	tp := &sync.Pool{}
	tp.New = func() any {
		return p.New(t)
	}
	p.pools[t] = tp
}

func (p *typePools) Put(t reflect.Type, x any) {
	if !UsePool {
		return
	}
	if r, ok := x.(Reset); ok {
		r.Reset()
	}

	p.pools[t].Put(x)
}
func (p *typePools) Get(t reflect.Type) any {
	if !UsePool {
		return p.New(t)
	}
	return p.pools[t].Get()
}
