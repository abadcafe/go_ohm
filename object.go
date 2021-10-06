package go_ohm

import (
	"reflect"
)

type abstractObject interface {
	renderValue() error
}

type object struct {
	name      string
	anonymous bool
	parent    *compoundObject
	*ObjectOptions

	// Reflected concrete type of the object. If original reflected type is
	// multiple level Pointer or Interface (A.K.A. indirect), here stored the
	// concrete type of the Pointer or Interface.
	typ reflect.Type

	// Reflected valid concrete value for the object. If typ is indirect, here
	// stored the eventual valid value. EX:
	//
	// 	var a *int = nil
	//	var b **int = &a
	//  var c interface{} = &b
	//  var d *interface{} = &c
	//
	// Then *value is reflect.ValueOf(a), which type is *int, and indirect is 3.
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

	abstractObject
}

var tagIdentifier = "go_ohm"
var rootObjectName = "__root_object"

func (o *object) isPlainObject() bool {
	return (o.typ.Kind() != reflect.Struct && o.typ.Kind() != reflect.Map) ||
		o.Json
}

func (o *object) isPromotedObject() bool {
	return o.typ.Kind() == reflect.Struct && o.anonymous && !o.Json &&
		o.Reference == "" && o.HashName == ""
}

func (o *object) createIndirectValues() {
	createIndirectValues(o.value, o.indirect)
}

func isIgnoredType(typ reflect.Type) bool {
	knd := typ.Kind()
	return knd == reflect.Chan || knd == reflect.Func ||
		knd == reflect.Invalid || knd == reflect.UnsafePointer ||
		knd == reflect.Interface
}

func isPrimitiveType(typ reflect.Type) bool {
	return (typ.Kind() == reflect.String) ||
		(typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8) ||
		(typ.Kind() >= reflect.Bool && typ.Kind() <= reflect.Complex128)
}

func advanceIndirectTypeAndValue(typ reflect.Type,
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

func createIndirectValues(v *reflect.Value, indirect int) {
	for i := 0; i < indirect; i++ {
		t := v.Type().Elem()
		p := reflect.New(t)
		v.Set(p)
		*v = v.Elem()
	}
}

func newObject(name string, parent *compoundObject, opts *ObjectOptions,
	typ reflect.Type, val *reflect.Value, indirect int, anon bool) (*object,
	error) {
	obj := &object{
		name:          name,
		anonymous:     anon,
		ObjectOptions: opts,
		typ:           typ,
		value:         val,
		indirect:      indirect,
		parent:        parent,
	}

	var err error
	if obj.isPlainObject() {
		_, err = newPlainObject(obj)
	} else if typ.Kind() == reflect.Struct {
		var co *compoundObject
		co, err = newCompoundObject(obj)
		if err != nil {
			return nil, err
		}
		_, err = newStructObject(co)
	} else if typ.Kind() == reflect.Map {
		var co *compoundObject
		co, err = newCompoundObject(obj)
		if err != nil {
			return nil, err
		}
		_, err = newMapObject(co)
	} else {
		err = newErrorUnsupportedObjectType(name)
	}
	if err != nil {
		return nil, err
	}

	return obj, nil
}
