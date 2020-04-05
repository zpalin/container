package container

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"sync"
)

const ConstructorName = "New"
const InitializerName = "Init"

type depContainer struct {
	types []reflect.Type
	impls map[reflect.Type]reflect.Type
	refs  map[reflect.Type]reflect.Value
	creating map[reflect.Type]interface{}
	hasBuilt bool
	wg sync.WaitGroup
}

type Runnable interface {
	Run()
}

type Container interface {
	// Register registers a component to be used to fulfill any requirements
	// for the specific type and any interfaces it implements.
	// If `comp` is a pointer, store as reference, otherwise use
	// passed concrete type and create new instance to fulfill requirements.
	Register(comps ...interface{})

	// RegisterAsInterface registers a component to be used to fulfill
	// interface requirement. `iface` must be an interface.
	// If `comp` is a pointer, store as reference, otherwise use
	// passed concrete type and create new instance to fulfill specfied
	// interface.
	RegisterAsInterface(iface interface{}, comp interface{})

	// Load returns a pointer to the specified type or implementor of specified interface
	// Panics if no type exists.
	Load(iface interface{}) interface{}

	// Load returns a pointer to the specified type or implementor of specified interface
	// and returns (nil, false) if none can be found.
	TryLoad(iface interface{}) (interface{}, bool)

	// Build iterates through registered components and attempts to
	// resolve their requirements. Panics if any constructor arguments
	// or public members cannot be satisfied with registered components.
	Build() Container

	// Run will take the supplied Runnable, inject dependencies into it,
	// and call .Run() on it.
	Run(r Runnable)

	// Run will take the supplied Runnable, inject dependencies into it,
	// and call .Run() on it.
	Exec(e interface{})

	// Async version of Run
	RunAsync(r Runnable)

	// Async version of Exec
	ExecAsync(e interface{})

	// Wait waits on any spawned background workers with RunAsync or ExecAsync
	Wait()
}

func (c *depContainer) Register(comps ...interface{}) {
	for _, cmp := range comps {
		typ := reflect.TypeOf(cmp)
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
			c.refs[typ] = reflect.ValueOf(cmp)
		}

		c.types = append(c.types, typ)
	}
}

func (c *depContainer) RegisterAsInterface(iface interface{}, comp interface{}) {
	ifaceTyp := reflect.TypeOf(iface).Elem()
	if ifaceTyp.Kind() != reflect.Interface {
		log.Fatal(fmt.Sprintf("%+v is not an interface", iface))
	}

	typ := reflect.TypeOf(comp)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		c.refs[typ] = reflect.ValueOf(comp)
	}

	if !reflect.New(typ).Type().Implements(ifaceTyp) {
		log.Fatal(fmt.Sprintf("%+v does not implement %+v", typ.Name(), ifaceTyp.Name()))
	}

	c.types = append(c.types, typ)
	c.impls[ifaceTyp] = typ
}

func constructorArgCount(typ reflect.Type) int {
	method, ok := typ.MethodByName("New")
	if !ok {
		return -1
	}
	return method.Type.NumIn()
}

func (c *depContainer) Build() Container {
	sort.Slice(c.types, func(i, j int) bool {
		return constructorArgCount(c.types[i]) < constructorArgCount(c.types[j])
	})

	// create registered types, skip those with refs already
	for _, typ := range c.types {
		_, ok := c.refs[typ]
		if ok {
			continue
		}
		c.createComponent(typ)
	}

	// initialize refs with initializers
	for _, val := range c.refs {
		if _, ok := val.Type().MethodByName(InitializerName); ok {
			val.MethodByName(InitializerName).Call(nil)
		}
	}
	c.hasBuilt = true
	return c
}

func (c *depContainer) verifyRegistry(typ reflect.Type) {
	for _, rt := range c.types {
		if rt == typ {
			return
		}
	}
	log.Fatal(fmt.Sprintf("no such dependency in registry: %+v %+v ", typ, typ.Kind()))
}

func (c *depContainer) findDep(bld, typ reflect.Type) reflect.Value {
	fmt.Printf("Typ.Kind(): %+v\n", typ.Kind())
	switch typ.Kind() {
	case reflect.Ptr:
		return c.getOrCreateComponent(typ.Elem())
	case reflect.Interface:
		return c.getOrCreateImpl(bld, typ)
	default:
		return c.getOrCreateComponent(typ).Elem()
	}
}

func (c *depContainer) getOrCreateComponent(typ reflect.Type) reflect.Value {
	c.verifyRegistry(typ)
	if comp, ok := c.refs[typ]; ok {
		return comp
	}
	return c.createComponent(typ)
}

func (c *depContainer) wireComponent(typ reflect.Type, val reflect.Value) reflect.Value {
	_, ok := val.Type().MethodByName(ConstructorName)
	// No constructor exists, set public members instead
	if !ok {
		for i := 0; i < val.Elem().NumField(); i++ {
			fld := val.Elem().Field(i)
			if !fld.CanSet() {
				continue
			}
			dep := c.findDep(typ, fld.Type())
			fld.Set(dep)
		}
		c.refs[typ] = val
		return val
	}

	// Gather required depContainer and call constructor
	method := val.MethodByName(ConstructorName)
	c.executeFunc(typ, method)
	c.refs[typ] = val
	return val
}

func (c *depContainer) createComponent(typ reflect.Type) reflect.Value {
	// Keep track of which records we're currently inputs the middle of creating
	// This prevents trying to use a type to satisfy it's own interface requirement
	// and also prevents cyclic references.
	c.creating[typ] = nil
	defer delete(c.creating, typ)

	val := reflect.New(typ)
	return c.wireComponent(typ, val)
}


func (c *depContainer) Load(iface interface{}) interface{} {
	ref, ok := c.TryLoad(iface); if !ok {
		log.Fatal(fmt.Sprintf("no instance of type found %+v", reflect.TypeOf(iface)))
	}
	return ref
}

// Load returns a pointer to the specified type or implementor of specified interface
// Returns (nil, false) if no type exists.
func (c *depContainer) TryLoad(iface interface{}) (interface{}, bool) {
	typ, ok := c.getNormalizedType(reflect.TypeOf(iface)); if !ok {
		return nil, false
	}
	ref, ok := c.refs[typ]; if !ok {
		return nil, false
	}
	return ref.Interface(), true
}

func (c *depContainer) getNormalizedType(typ reflect.Type) (reflect.Type, bool) {
	switch typ.Kind() {
	case reflect.Ptr:
		return c.getNormalizedType(typ.Elem())
	case reflect.Interface:
		typ, ok := c.impls[typ]
		return typ, ok
	}
	return typ, true
}

func (c *depContainer) findImplementor(bld, iface reflect.Type) reflect.Type {
	typ, ok := c.impls[iface]
	if ok {
		if _, creating := c.creating[typ]; !creating {
			return typ
		}
	}

	for _, typ = range c.types {
		ptrTyp := reflect.New(typ).Type()
		if ptrTyp.Implements(iface) {
			if _, creating := c.creating[typ]; creating {
				continue
			}
			c.impls[iface] = typ
			return typ
		}
	}
	panic(fmt.Sprintf("%+v implementor not found, required by %+v", iface.Name(), bld.Name()))
}

func (c *depContainer) getOrCreateImpl(bld, iface reflect.Type) reflect.Value {
	implTyp := c.findImplementor(bld, iface)
	return c.getOrCreateComponent(implTyp)
}

func (c *depContainer) executeFunc(typ reflect.Type, fn reflect.Value) {
	argCount := fn.Type().NumIn()
	inputs := make([]reflect.Value, argCount)
	for i := 0; i < argCount; i++ {
		in := fn.Type().In(i)
		inputs[i] = c.findDep(typ, in)
	}
	fn.Call(inputs)
}

func (c *depContainer) Exec(e interface{}) {
	if !c.hasBuilt {
		c.Build()
	}
	val := reflect.ValueOf(e)
	c.executeFunc(val.Type(), val)
}

func (c *depContainer) Run(r Runnable) {
	if !c.hasBuilt {
		c.Build()
	}

	val := reflect.ValueOf(r)
	c.wireComponent(val.Elem().Type(), val)
	r.Run()
}

func (c *depContainer) RunAsync(r Runnable) {
	c.wg.Add(1)
	go func() {
		c.Run(r)
		c.wg.Done()
	}()
}

func (c *depContainer) ExecAsync(e interface{}) {
	c.wg.Add(1)
	go func() {
		c.Exec(e)
		c.wg.Done()
	}()
}

func (c *depContainer) Wait() {
	c.wg.Wait()
}

func New() Container {
	c := &depContainer{
		impls: make(map[reflect.Type]reflect.Type),
		refs:  make(map[reflect.Type]reflect.Value),
		creating:  make(map[reflect.Type]interface{}),
		hasBuilt: false,
	}
	c.RegisterAsInterface((*Container)(nil), c)
	return c
}
