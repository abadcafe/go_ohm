package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
)

type mapObject struct {
	*object

	// Reflected type of index, for map only.
	indexTyp reflect.Type

	reply [][]byte
}

func (o *mapObject) doRedisHMGet(conn redis.Conn, prefix string) error {
	return nil
}

func (o *mapObject) renderValue() error {
	return nil
}

func (o *mapObject) complete() error {
	o.indexTyp = o.typ.Key()
	switch o.indexTyp.Kind() {
	case reflect.Int:
	case reflect.Int8:
	case reflect.Int16:
	case reflect.Int32:
	case reflect.Int64:
	case reflect.Uint:
	case reflect.Uint8:
	case reflect.Uint16:
	case reflect.Uint32:
	case reflect.Uint64:
	case reflect.String:
	default:
		return NewErrorUnsupportedObjectType(o.name)
	}

	return nil
}

func completeMapObject(bo *object) error {
	obj := &mapObject{object: bo}
	bo.concreteObject = obj
	return obj.complete()
}
