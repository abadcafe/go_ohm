package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
)

type concreteObject interface {
	renderValue() error
}

type compoundObject interface {
	concreteObject
	getDescendants(objList *[]*object)
	doRedisHMGet(conn redis.Conn, prefix string) error
}

type objectOptions struct {
	hashKey     string
	hashField   string
	hashPrefix  string
	reference   string
	nonJson     bool
	elemNonJson bool
}

type object struct {
	name   string
	parent *object
	*objectOptions

	// Reflected concrete type of the object. If original reflected type is
	// multiple level Pointer or Interface (A.K.A. indirect), here stored the
	// concrete type of the Pointer or Interface.
	typ reflect.Type

	// Reflected valid concrete value for the object. If typ is indirect, here
	// stored the eventual valid value. EX:
	//
	// 	var b *int = nil
	//	var a **int = &b
	//
	// Then *value is reflect.ValueOf(b), which type is *int, and indirect is 1.
	//
	// And sometimes value can be nil. EX, there are two structs and a variable:
	//
	// struct A{
	//     A int
	// }
	//
	// struct B {
	//     B **A
	// }
	//
	// var s struct B
	//
	// Now the object which represented s.B.A, has a nil value field.
	value    *reflect.Value
	indirect int

	concreteObject
}

var tagIdentifier = "go_ohm"
var rootObjectName = "root object"

func (o *object) isPlainObject() bool {
	return (o.typ.Kind() != reflect.Struct && o.typ.Kind() != reflect.Map) ||
		!o.nonJson
}

func (o *object) isTiledObject() bool {
	return o.typ.Kind() == reflect.Struct && o.name == "" && o.nonJson &&
		o.reference == "" && o.hashKey == ""
}

func (o *object) createIndirectValues() {
	if o.indirect <= 0 {
		return
	}

	v := o.value
	for i := 0; i < o.indirect; i++ {
		t := v.Type().Elem()
		p := reflect.New(t)
		v.Set(p)
		*v = p.Elem()
	}
}

func isIgnoredType(typ reflect.Type) bool {
	knd := typ.Kind()
	return knd == reflect.Chan || knd == reflect.Func ||
		knd == reflect.Invalid || knd == reflect.UnsafePointer ||
		knd == reflect.Interface
}

func isPrimitiveType(typ reflect.Type) bool {
	return (typ.Kind() >= reflect.Bool && typ.Kind() <= reflect.Complex128) ||
		(typ.Kind() == reflect.String) ||
		(typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8) ||
		(typ.Kind() == reflect.Array && typ.Elem().Kind() == reflect.Uint8)
}

func objectConcreteType(typ reflect.Type,
	val *reflect.Value) (reflect.Type, *reflect.Value, int) {
	if val != nil && val.IsValid() {
		for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
			if val.IsZero() {
				break
			}

			v := val.Elem()
			val = &v
		}

		typ = val.Type()
	}

	indirect := 0
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		indirect++
	}

	return typ, val, indirect
}

func buildObject(name string, parent *object, opts *objectOptions,
	typ reflect.Type, val *reflect.Value, indirect int) (*object, error) {
	obj := &object{
		name:          name,
		objectOptions: opts,
		typ:           typ,
		value:         val,
		indirect:      indirect,
		parent:        parent,
	}

	var err error
	if obj.isPlainObject() {
		err = completePlainObject(obj)
	} else if typ.Kind() == reflect.Struct {
		err = completeStructObject(obj)
	} else if typ.Kind() == reflect.Map {
		err = completeMapObject(obj)
	} else {
		err = NewErrorUnsupportedObjectType(name)
	}
	if err != nil {
		return nil, err
	}

	return obj, nil
}
