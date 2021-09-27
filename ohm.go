package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
)

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

func Load(conn redis.Conn, ns string, key string, data interface{}) error {
	name := rootObjectName

	t := reflect.TypeOf(data)
	if t == nil {
		return NewErrorUnsupportedObjectType(name)
	}

	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return NewErrorUnsupportedObjectType(name)
	}

	typ, val, indirect := advanceIndirectTypeAndValue(t, &v)
	if isIgnoredType(typ) {
		// do not support those types, skip.
		return NewErrorUnsupportedObjectType(name)
	}

	opts := &objectOptions{
		hashPrefix:  "",
		hashKey:     key,
		hashField:   "",
		reference:   "",
		nonJson:     true,
		elemNonJson: false,
	}
	obj, err := buildObject(name, nil, opts, typ, val, indirect)
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

func Save(key string, obj interface{}) error {
	return nil
}
