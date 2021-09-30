package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
)

type ObjectOptions struct {
	hashKey     string
	hashField   string
	hashPrefix  string
	reference   string
	json        bool
	elemNonJson bool
}

func doLoadCommands(conn redis.Conn, ns string, obj *compoundObject) error {
	var objs []*compoundObject
	obj.getDescendants(&objs)

	for _, o := range objs {
		err := o.doRedisHMGet(conn, ns)
		if err != nil {
			return err
		}
	}

	return nil
}

func Load(conn redis.Conn, ns string, opts *ObjectOptions, i interface{}) error {
	name := rootObjectName

	t := reflect.TypeOf(i)
	if t == nil {
		return NewErrorUnsupportedObjectType(name)
	}

	v := reflect.ValueOf(i)
	if !v.IsValid() {
		return NewErrorUnsupportedObjectType(name)
	}

	typ, val, indirect := advanceIndirectTypeAndValue(t, &v)
	if isIgnoredType(typ) {
		// do not support those types, skip.
		return NewErrorUnsupportedObjectType(name)
	}

	obj, err := newObject(name, nil, opts, typ, val, indirect, false)
	if err != nil {
		return err
	} else if obj.isPlainObject() {
		return NewErrorUnsupportedObjectType(name)
	}

	err = doLoadCommands(conn, ns, obj.abstractObject.(*compoundObject))
	if err != nil {
		return err
	}

	return obj.renderValue()
}

func Save(conn redis.Conn, ns string, opts *ObjectOptions, i interface{}) error {
	return nil
}
