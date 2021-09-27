package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
)

type mapObject struct {
	*compoundObject

	// Reflected type of index, for map only.
	indexTyp reflect.Type

	// redis reply of a redis hash.
	reply map[string]string
}

func (o *mapObject) getDescendants(objList *[]*compoundObject) {
	*objList = append(*objList, o.compoundObject)
}

func (o *mapObject) doRedisHMGet(conn redis.Conn, prefix string) error {
	key, err := o.genRedisHashKey(prefix)
	if err != nil {
		return err
	}

	rep, err := redis.StringMap(conn.Do("HGETALL", key))
	if err != nil {
		return NewErrorRedisCommandsFailed(o.name, err)
	}

	o.reply = rep
	return nil
}

func (o *mapObject) renderValue() error {
	o.createIndirectValues()

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

func completeMapObject(o *object) (*mapObject, error) {
	co, err := completeCompoundObject(o)
	if err != nil {
		return nil, err
	}

	obj := &mapObject{compoundObject: co}
	obj.abstractCompoundObject = obj
	err = obj.complete()
	if err != nil {
		return nil, err
	}

	return obj, nil
}
