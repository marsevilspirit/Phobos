package client

import "context"

type PluginContainer interface {
	Add(plugin Plugin)
	Remove(plugin Plugin)
	All() []Plugin

	DoPreCall(ctx context.Context, servicePath, serviceMethod string, args any) error
	DoPostCall(ctx context.Context, servicePath, serviceMethod string, args any, reply any, err error) error
}

type pluginContainer struct {
	plugins []Plugin
}

type Plugin any

type (
	// PreCallPlugin is invoked before the client calls a server.
	PreCallPlugin interface {
		DoPreCall(ctx context.Context, servicePath, serviceMethod string, args any) error
	}

	// PostCallPlugin is invoked after the client calls a server.
	PostCallPlugin interface {
		DoPostCall(ctx context.Context, servicePath, serviceMethod string, args any, reply any, err error) error
	}
)

func (p *pluginContainer) Add(plugin Plugin) {
	p.plugins = append(p.plugins, plugin)
}

func (p *pluginContainer) Remove(plugin Plugin) {
	if p.plugins == nil {
		return
	}

	var plugins []Plugin

	for _, pp := range p.plugins {
		if pp != plugin {
			plugins = append(plugins, pp)
		}
	}

	p.plugins = plugins
}

func (p *pluginContainer) All() []Plugin {
	return p.plugins
}

// DoPreCall executes before call
func (p *pluginContainer) DoPreCall(ctx context.Context, servicePath, serviceMethod string, args any) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PreCallPlugin); ok {
			err := plugin.DoPreCall(ctx, servicePath, serviceMethod, args)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DoPostCall executes after call
func (p *pluginContainer) DoPostCall(ctx context.Context, servicePath, serviceMethod string, args any, reply any, err error) error {
	for i := range p.plugins {
		if plugin, ok := p.plugins[i].(PostCallPlugin); ok {
			err = plugin.DoPostCall(ctx, servicePath, serviceMethod, args, reply, err)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
