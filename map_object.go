package go_ohm

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/gomodule/redigo/redis"
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

func (o *mapObject) doRedisLoad(conn redis.Conn, ns string) error {
	key, err := o.genRedisKey(ns)
	if err != nil {
		return err
	}

	rep, err := redis.StringMap(conn.Do("HGETALL", key))
	if err != nil {
		return newErrorRedisCommandFailed(o.name, err)
	}

	o.reply = rep
	return nil
}

func (o *mapObject) genHashFieldValuePairs() ([]interface{}, error) {
	var cmdArgs []interface{}

	iter := o.value.MapRange()
	for iter.Next() {
		k := fmt.Sprint(iter.Key().Interface())

		vv := iter.Value()
		v, err := jsonMarshalValue(&vv)
		if err != nil {
			return nil, newErrorJsonFailed(o.name, err)
		}

		cmdArgs = append(cmdArgs, k, string(v))
	}

	return cmdArgs, nil
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
		i, err := strconv.ParseInt(s, 10, 64)
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
		u, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, err
		}
		v = reflect.ValueOf(u).Convert(o.indexTyp)

	case reflect.String:
		v = reflect.ValueOf(s)
	}

	return &v, nil
}

func (o *mapObject) renderValue() error {
	o.createIndirectValues()
	if o.value.IsNil() {
		o.value.Set(reflect.MakeMap(o.value.Type()))
	}

	if o.ElemNonJson {
		return newErrorUnsupportedObjectType(o.name)
	}

	for rk, rv := range o.reply {
		k, err := o.newIndexValue(rk)
		if err != nil {
			return newErrorUnsupportedObjectType(o.name)
		}

		// use reflect.New() to create a pointer to map's element, otherwise the
		// element was unaddressable and unsettable.
		v := o.value.MapIndex(*k)
		p := reflect.New(o.typ.Elem())
		if v.IsValid() {
			p.Elem().Set(v)
		}
		v = p.Elem()

		vt, vv, vi := advanceIndirectTypeAndValue(v.Type(), &v)
		if isIgnoredType(vt) {
			return newErrorUnsupportedObjectType(o.name)
		}
		createIndirectValues(vv, vi)

		err = jsonUnmarshalValue([]byte(rv), vv)
		if err != nil {
			return newErrorJsonFailed(o.name, err)
		}

		o.value.SetMapIndex(*k, v)
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
		return newErrorUnsupportedObjectType(o.name)
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
