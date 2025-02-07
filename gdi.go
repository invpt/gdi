package gdi

import (
	"reflect"
)

type RegistrationHook = func(any)

type ContainerBuilder struct {
	hooks        []func(any)
	constructors map[reflect.Type]reflect.Value
	values       map[reflect.Type]reflect.Value
}

type Container struct {
	items map[reflect.Type]reflect.Value
}

func Builder() *ContainerBuilder {
	return &ContainerBuilder{
		constructors: map[reflect.Type]reflect.Value{},
		values:       map[reflect.Type]reflect.Value{},
	}
}

func (c *ContainerBuilder) RegisterHook(h RegistrationHook) {
	c.hooks = append(c.hooks, h)
}

func (c *ContainerBuilder) Register(constructor any) {
	value := reflect.ValueOf(constructor)
	if value.Kind() != reflect.Func {
		c.values[value.Type()] = value
	}
	c.constructors[value.Type().Out(0)] = value
}

func (c *ContainerBuilder) Build() *Container {
	visited := map[reflect.Type]struct{}{}
	for len(c.constructors) != 0 {
		for t, _ := range c.constructors {
			c.register(t, visited)
			break
		}
	}
	return &Container{items: c.values}
}

func (c *ContainerBuilder) register(
	t reflect.Type,
	visited map[reflect.Type]struct{},
) reflect.Value {
	if constructed, ok := c.values[t]; ok {
		return constructed
	} else if _, alreadyVisited := visited[t]; alreadyVisited {
		panic("loop")
	}
	visited[t] = struct{}{}

	constructor, ok := c.constructors[t]
	if !ok {
		panic("Requested an unregistered service")
	}

	args := make([]reflect.Value, 0, constructor.Type().NumIn())
	for i := range constructor.Type().NumIn() {
		args = append(args, c.register(constructor.Type().In(i), visited))
	}

	output := constructor.Call(args)
	if len(output) != 1 {
		panic("Constructors must return one thing")
	}

	c.values[output[0].Type()] = output[0]
	delete(c.constructors, constructor.Type().Out(0))

	for _, hook := range c.hooks {
		hook(output[0].Interface())
	}

	return output[0]
}
