package server

import (
	"context"
	"net"

	"github.com/marsevilspirit/m_RPC/errors"
	"github.com/marsevilspirit/m_RPC/protocol"
)

type PluginContainer interface {
	Add(plugin Plugin)
	Remove(plugin Plugin)
	All() []Plugin

	DoRegister(name string, rcvr interface{}, metadata string) error

	DoPostConnAccept(net.Conn) (net.Conn, bool)

	DoPreReadRequest(ctx context.Context) error
	DoPostReadRequest(ctx context.Context, r *protocol.Message, e error) error

	DoPreWriteResponse(context.Context, *protocol.Message) error
	DoPostWriteResponse(context.Context, *protocol.Message, *protocol.Message, error) error
}

type Plugin interface{}

type (
	RegisterPlugin interface {
		Register(name string, rcvr interface{}, metadata string) error
	}

	PostConnAcceptPlugin interface {
		HandleConAccept(net.Conn) (net.Conn, bool)
	}

	PreReadRequestPlugin interface {
		PreReadRequest(ctx context.Context) error
	}

	PostReadRequestPlugin interface {
		PostReadRequest(ctx context.Context, r *protocol.Message, e error) error
	}

	PreWriteResponsePlugin interface {
		PreWriteResponse(context.Context, *protocol.Message) error
	}

	PostWriteResponsePlugin interface {
		PostWriteResponse(context.Context, *protocol.Message, *protocol.Message, error) error
	}
)

type pluginContainer struct {
	plugins []Plugin
}

func (p *pluginContainer) Remove(plugin Plugin) {
	if p.plugins == nil {
		return
	}

	var plugins []Plugin
	for _, p := range p.plugins {
		if p != plugin {
			plugins = append(plugins, p)
		}
	}

	p.plugins = plugins
}

func (p *pluginContainer) All() []Plugin {
	return p.plugins
}

func (p *pluginContainer) DoRegister(name string, rcvr interface{}, metadata string) error {
	var es []error
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(RegisterPlugin); ok {
			err := plugin.Register(name, rcvr, metadata)
			if err != nil {
				es = append(es, err)
			}
		}
	}

	if len(es) > 0 {
		return errors.NewMultiError(es)
	}

	return nil
}

func (p *pluginContainer) DoPostConnAccept(conn net.Conn) (net.Conn, bool) {
	var flag bool
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostConnAcceptPlugin); ok {
			conn, flag = plugin.HandleConAccept(conn)
			if !flag {
				conn.Close()
				return conn, false
			}
		}
	}

	return conn, true
}

func (p *pluginContainer) DoPreReadRequest(ctx context.Context) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PreReadRequestPlugin); ok {
			err := plugin.PreReadRequest(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *pluginContainer) DoPostReadRequest(ctx context.Context, r *protocol.Message, e error) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostReadRequestPlugin); ok {
			err := plugin.PostReadRequest(ctx, r, e)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *pluginContainer) DoPreWriteResponse(ctx context.Context, req *protocol.Message) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PreWriteResponsePlugin); ok {
			err := plugin.PreWriteResponse(ctx, req)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *pluginContainer) DoPostWriteResponse(ctx context.Context, req *protocol.Message, resp *protocol.Message, e error) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostWriteResponsePlugin); ok {
			err := plugin.PostWriteResponse(ctx, req, resp, e)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
