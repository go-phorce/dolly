package container

import (
	"go.uber.org/dig"
)

// Container provides IoC interface
type Container interface {
	Provide(constructor interface{}, opts ...ProvideOption) error
	Invoke(function interface{}, opts ...InvokeOption) error
}

// ProvideOption specifies options for Provide interface
type ProvideOption dig.ProvideOption

// InvokeOption specifies options for Invoke interface
type InvokeOption dig.InvokeOption

type proxy struct {
	*dig.Container
}

// New returns new container
func New() Container {
	return &proxy{dig.New()}
}

func (p *proxy) Provide(constructor interface{}, opts ...ProvideOption) error {
	list := make([]dig.ProvideOption, len(opts))
	for i, op := range opts {
		list[i] = op.(dig.ProvideOption)
	}

	return p.Container.Provide(constructor, list...)
}

func (p *proxy) Invoke(function interface{}, opts ...InvokeOption) error {
	list := make([]dig.InvokeOption, len(opts))
	for i, op := range opts {
		list[i] = op.(dig.InvokeOption)
	}
	return p.Container.Invoke(function, list...)
}
