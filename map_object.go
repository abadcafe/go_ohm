package go_ohm

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"reflect"
	"strconv"
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
		return NewErrorRedisCommandFailed(o.name, err)
	}

	o.reply = rep
	return nil
}

func (o *mapObject) newIndexValue(s string) (*reflect.Value, error) {
	var v reflect.Value

	switch o.indexTyp.Kind() {
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		i, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}

		v = reflect.ValueOf(i).Convert(o.indexTyp)

	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		fallthrough
	case reflect.Uintptr:
		i, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}

		v = reflect.ValueOf(i).Convert(o.indexTyp)

	case reflect.String:
		v = reflect.ValueOf(s)
	}

	return &v, nil
}

func (o *mapObject) renderValue() error {
	o.createIndirectValues()

	if o.elemNonJson {
		return NewErrorUnsupportedObjectType(o.name)
	}

	for k, v := range o.reply {
		rk, err := o.newIndexValue(k)
		if err != nil {
			return NewErrorUnsupportedObjectType(o.name)
		}

		rv := o.value.MapIndex(*rk)
		if !rv.IsValid() {
			rv = reflect.New(o.typ.Elem())
		}

		vt, vv, vi := advanceIndirectTypeAndValue(rv.Type(), &rv)
		if isIgnoredType(vt) {
			return NewErrorUnsupportedObjectType(o.name)
		}

		createIndirectValues(vv, vi)
		err = json.Unmarshal([]byte(v), vv.Addr().Interface())
		if err != nil {
			return NewErrorJsonFailed(o.name, err)
		}
	}

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

func newMapObject(co *compoundObject) (*mapObject, error) {
	obj := &mapObject{compoundObject: co}
	obj.abstractCompoundObject = obj
	err := obj.complete()
	if err != nil {
		return nil, err
	}

	return obj, nil
}
