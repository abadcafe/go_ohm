// Package go_ohm is for "Object to redis hash" mapping.
//
// It use go's struct tag to indicate how to map a struct field to a redis hash
// field. And includes reference directive, which likes SQL's ForeignKey.
//
// See `ObjectOptions` to know all usable struct tags.
package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
)

// ObjectOptions is options for a struct field. Every struct field is a `Object`
// internally.
//
// This type exported mainly for customize Load() and Save()'s argument `i`'s
// behaviors, because `i` is the "root object" and is not a field of certain
// struct, so can't use struct tag.
//
// Every option has a corresponding struct tag option, in sneak case. For
// instance:
//   type A struct {
//     A int `go_ohm:"hash_name=a"`
//   }
// The struct tag are parsed as a `ObjectOptions` internally, and customized
// field `A`'s mapping result.
type ObjectOptions struct {
	// Redis hash's name, for map and struct only. Default is field name.
	HashName string

	// Redis hash's field, for primitive types and jsonified compound types,
	// Default is field name.
	HashField string

	// Prefix of hash field. Default is field type name.
	HashPrefix string

	// Refer to other field. If presented, the hash name is referred field
	// value.
	Reference string

	// Jsonify the struct field, and store as a hash field. This option
	// corresponded two struct tag options: "json" and "non_json". For compound
	// types, includes slice(except byte slice), array, map and struct, default
	// is "json", and for other types default is "non_json".
	Json        bool

	// Don't Jsonify elements of map. Only for field which type is map. default
	// is jsonify all types.
	ElemNonJson bool
}

// Load data struct from redis hash.
//
// `conn` is a redis connection created by external package `redigo`. See
// https://github.com/gomodule/redigo.
//
// `ns` is namespace to classify hash keys. It is represented as prefix of hash
// keys.
//
// `opts` specified how to deal with data struct in `i`. See `ObjectOptions`.
//
// `i` is data struct, currently it supports struct pointer, map, and map
// pointer. The map key must be int, uint or string.
//
// It returns `error` while failed.
func Load(conn redis.Conn, ns string, opts *ObjectOptions, i interface{}) error {
	objs, err := genObjectList(i, opts)
	if err != nil {
		return err
	}

	err = doLoadCommands(conn, ns, objs)
	if err != nil {
		return err
	}

	// first object is the root object.
	return objs[0].renderValue()
}

// Save data struct to redis hash. See Load() for argument explanation.
func Save(conn redis.Conn, ns string, opts *ObjectOptions, i interface{}) error {
	objs, err := genObjectList(i, opts)
	if err != nil {
		return err
	}

	return doSaveCommands(conn, ns, objs)
}

func genObjectList(i interface{}, opts *ObjectOptions) ([]*compoundObject, error) {
	name := rootObjectName

	t := reflect.TypeOf(i)
	if t == nil {
		return nil, newErrorUnsupportedObjectType(name)
	}

	v := reflect.ValueOf(i)
	if !v.IsValid() {
		return nil, newErrorUnsupportedObjectType(name)
	}

	typ, val, indirect := advanceIndirectTypeAndValue(t, &v)
	if isIgnoredType(typ) {
		// do not support those types, skip.
		return nil, newErrorUnsupportedObjectType(name)
	}

	obj, err := newObject(name, nil, opts, typ, val, indirect, false)
	if err != nil {
		return nil, err
	} else if obj.isPlainObject() {
		return nil, newErrorUnsupportedObjectType(name)
	}

	var objs []*compoundObject
	rootObj := obj.abstractObject.(*compoundObject)
	rootObj.getDescendants(&objs)

	return objs, nil
}

func doLoadCommands(conn redis.Conn, ns string, objs []*compoundObject) error {
	for _, o := range objs {
		err := o.doRedisLoad(conn, ns)
		if err != nil {
			return err
		}
	}

	return nil
}

func doSaveCommands(conn redis.Conn, ns string, objs []*compoundObject) error {
	for _, o := range objs {
		err := o.doRedisSave(conn, ns)
		if err != nil {
			return err
		}
	}

	return nil
}
