package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
)

type ObjectOptions struct {
	hashName    string
	hashField   string
	hashPrefix  string
	reference   string
	json        bool
	elemNonJson bool
}

func genObjectList(i interface{}, opts *ObjectOptions) ([]*compoundObject, error) {
	name := rootObjectName

	t := reflect.TypeOf(i)
	if t == nil {
		return nil, NewErrorUnsupportedObjectType(name)
	}

	v := reflect.ValueOf(i)
	if !v.IsValid() {
		return nil, NewErrorUnsupportedObjectType(name)
	}

	typ, val, indirect := advanceIndirectTypeAndValue(t, &v)
	if isIgnoredType(typ) {
		// do not support those types, skip.
		return nil, NewErrorUnsupportedObjectType(name)
	}

	obj, err := newObject(name, nil, opts, typ, val, indirect, false)
	if err != nil {
		return nil, err
	} else if obj.isPlainObject() {
		return nil, NewErrorUnsupportedObjectType(name)
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

func Save(conn redis.Conn, ns string, opts *ObjectOptions, i interface{}) error {
	objs, err := genObjectList(i, opts)
	if err != nil {
		return err
	}

	return doSaveCommands(conn, ns, objs)
}
