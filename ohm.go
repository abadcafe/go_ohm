package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
)

func doLoadCommands(conn redis.Conn, prefix string, obj *object) error {
	var cos []*object
	co := obj.concreteObject.(compoundObject)
	co.getDescendants(&cos)

	for _, o := range cos {
		err := o.concreteObject.(compoundObject).doRedisHMGet(conn, prefix)
		if err != nil {
			return err
		}
	}

	return nil
}

func Load(conn redis.Conn, prefix string, key string, data interface{}) error {
	name := rootObjectName

	typ := reflect.TypeOf(data)
	if typ == nil {
		return NewErrorUnsupportedObjectType(name)
	}

	val0 := reflect.ValueOf(data)
	if !val0.IsValid() {
		return NewErrorUnsupportedObjectType(name)
	}

	typ, val, indirect := objectConcreteType(typ, &val0)
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
	}

	err = doLoadCommands(conn, prefix, obj)
	if err != nil {
		return err
	}

	return obj.renderValue()
}

func Save(key string, obj interface{}) error {
	return nil
}
