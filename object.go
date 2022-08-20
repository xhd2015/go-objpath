package objpath

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"unicode"
)

type Object interface {
	Value() interface{}
	Method(name string) (method interface{}, ok bool)
}

type Primitive interface {
	Object
	StrValue() string
}
type Composite interface {
	Object
	ChildrenLen() int
	RangeChildren(func(key string, child Object) bool)
	GetChild(key string) (child Object, ok bool)
}

// don't respect JSONMarshaler and JSONUnmarshaler interface
// NOTE: v cannot be reflect.Value
func NewObject(v interface{}) Object {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		return &List{
			base: base{
				rv: rv,
			},
		}
	case reflect.Map:
		return &Map{
			base: base{
				rv: rv,
			},
		}
	case reflect.Struct:
		return &Struct{
			base: base{
				rv: rv,
			},
		}
	case reflect.String:
		iv := rv.Interface()
		return NewPrimitve(iv, iv.(string))
	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64:
		return NewPrimitve(rv.Interface(), fmt.Sprint(rv.Interface()))
	case reflect.Func, reflect.Chan:
		return nil
	default:
		return nil
	}
}

type Struct struct {
	base
	m    *SortedMap // map[string]Object
	once sync.Once
}

// Value implements Object
func (c *Struct) Value() interface{} {
	return c.rv.Interface()
}

// ChildrenLen implements Object
func (c *Struct) ChildrenLen() int {
	return c.getChildren().Len()
}

// GetChild implements Object
func (c *Struct) GetChild(key string) (child Object, ok bool) {
	v, ok := c.getChildren().GetOK(key)
	if !ok {
		return nil, false
	}
	return v.(Object), true
}

// RangeChildren implements Object
func (c *Struct) RangeChildren(fn func(key string, child Object) bool) {
	c.getChildren().Range(func(key string, val interface{}) bool {
		return fn(key, val.(Object))
	})
}
func (c *Struct) getChildren() *SortedMap {
	c.once.Do(func() {
		c.m = NewSortedMap(c.rv.NumField())
		TraverseStruct(c.rv, func(value reflect.Value, field, parentField *reflect.StructField) {
			// about efficiency: given that overlapping does not happen so frequently
			// we can just compute for each Object, they may be later discared
			// due to name override
			if field.Name != "" && unicode.ToUpper(rune(field.Name[0])) == rune(field.Name[0]) {
				c.m.Set(field.Name, NewObject(value.Interface()))
			}
		})
	})
	return c.m
}

var _ Object = ((*Struct)(nil))

type Map struct {
	base

	m    map[string]Object // map[string]Object
	once sync.Once
}

var _ Object = ((*Map)(nil))

// ChildrenLen implements Object
func (c *Map) ChildrenLen() int {
	return c.rv.Len()
}

// GetChild implements Object
func (c *Map) GetChild(key string) (child Object, ok bool) {
	child, ok = c.getChildren()[key]
	return
}

// RangeChildren implements Object
func (c *Map) RangeChildren(fn func(key string, child Object) bool) {
	for k, v := range c.getChildren() {
		if !fn(k, v) {
			return
		}
	}
}

// Value implements Object
func (c *Map) Value() interface{} {
	return c.rv.Interface()
}

func (c *Map) getChildren() map[string]Object {
	c.once.Do(func() {
		c.m = make(map[string]Object, c.rv.Len())
		for it := c.rv.MapRange(); it.Next(); {
			c.m[fmt.Sprint(it.Key())] = NewObject(it.Value().Interface())
		}
	})
	return c.m
}

type List struct {
	base
	list []Object
	once sync.Once
}

// Value implements Object
func (c *List) Value() interface{} {
	return c.rv.Interface()
}

var _ Object = ((*List)(nil))

// ChildrenLen implements Object
func (c *List) ChildrenLen() int {
	return c.rv.Len()
}

// GetChild implements Object
func (c *List) GetChild(key string) (child Object, ok bool) {
	i, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return nil, false
	}
	if i < 0 {
		return nil, false
	}
	children := c.getChildren()
	if i >= int64(len(children)) {
		return nil, false
	}
	return children[i], true
}

// IsPrimitive implements Object
func (c *List) IsPrimitive() bool {
	return false
}

// RangeChildren implements Object
func (c *List) RangeChildren(fn func(key string, child Object) bool) {
	for i, child := range c.getChildren() {
		if !fn(strconv.FormatInt(int64(i), 10), child) {
			return
		}
	}
}
func (c *List) getChildren() []Object {
	c.once.Do(func() {
		n := c.rv.Len()
		c.list = make([]Object, n)
		for i := 0; i < n; i++ {
			c.list[i] = NewObject(c.rv.Index(i).Interface())
		}
	})
	return c.list
}

type base struct {
	rv reflect.Value
}

func (c *base) String() string {
	return fmt.Sprint(c.rv.Interface())
}
func (c *base) Method(name string) (method interface{}, ok bool) {
	return MethodOf(c.rv, name)
}

type SPrimitive struct {
	val interface{}
	str string
}

var _ Primitive = ((*SPrimitive)(nil))

func NewPrimitve(val interface{}, str string) Primitive {
	return &SPrimitive{
		val: val,
		str: str,
	}
}

// Value implements Object
func (c *SPrimitive) Value() interface{} {
	return c.val
}

// Value implements Object
func (c *SPrimitive) StrValue() string {
	return c.str
}
func (c *SPrimitive) String() string {
	return c.str
}

func (c *SPrimitive) Method(name string) (method interface{}, ok bool) {
	if c.val == nil {
		return nil, false
	}
	return MethodOf(reflect.ValueOf(c.val), name)
}

// c or *c
func MethodOf(v reflect.Value, name string) (method interface{}, ok bool) {
	if !v.IsValid() {
		return
	}
	fn := v.MethodByName(name)
	if fn.IsValid() {
		method = fn.Interface()
		ok = true
		return
	}
	// check ptr
	// if c.val is already a pointer
	// then no try. since only non-pointer type can
	// define a pointer receiver
	if v.Kind() == reflect.Ptr {
		return
	}
	pv := reflect.New(v.Type())
	pv.Elem().Set(v)
	fn = pv.MethodByName(name)
	if fn.IsValid() {
		method = fn.Interface()
		ok = true
		return
	}
	return
}

// CallFn make calls to fn
func CallFn(fn interface{}, args []string) (res []interface{}, err error) {
	fnv := reflect.ValueOf(fn)
	fnType := fnv.Type()

	argN := len(args)
	n := fnType.NumIn()
	variadic := fnType.IsVariadic()

	// check length
	if !variadic {
		if argN != n {
			err = fmt.Errorf("expecting %d args, actual: %d", n, argN)
			return
		}
	} else {
		if argN < n-1 {
			err = fmt.Errorf("expecting at least %d args, actual: %d", n-1, argN)
			return
		}
	}
	var v reflect.Value
	argsV := make([]reflect.Value, 0, len(args))

	for i := 0; i < argN; i++ {
		var t reflect.Type
		if i < n-1 || !variadic {
			t = fnType.In(i)
		} else {
			t = fnType.In(n - 1)
		}
		v, err = createArg(args[i], t)
		if err != nil {
			err = fmt.Errorf("parsing arg %d: %v", i, err)
			return
		}
		argsV = append(argsV, v)
	}

	vals := fnv.Call(argsV)
	if len(vals) == 0 {
		return
	}
	res = make([]interface{}, 0, len(vals))
	for _, val := range vals {
		res = append(res, val.Interface())
	}
	lastVal := vals[len(vals)-1]
	if lastVal.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		lastErr := res[len(res)-1]
		res = res[:len(res)-1]
		if lastErr != nil {
			err = lastErr.(error)
			return
		}
	}
	return
}

// we can parse int

func createArg(arg string, typ reflect.Type) (reflect.Value, error) {
	v := reflect.New(typ)
	switch typ.Kind() {
	case reflect.Int:
		i, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("parsing number: %v %v", arg, err)
		}
		v.Elem().Set(reflect.ValueOf(int(i)))
	case reflect.String:
		v.Elem().Set(reflect.ValueOf(arg))
	default:
		return reflect.Value{}, fmt.Errorf("unrecognized type:%v %v", arg, typ)
	}
	return v.Elem(), nil
}
