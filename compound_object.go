package go_ohm

import (
	"strings"

	"github.com/gomodule/redigo/redis"
)

type abstractCompoundObject interface {
	abstractObject
	getDescendants(objList *[]*compoundObject)
	doRedisLoad(conn redis.Conn, ns string) error
	genHashFieldValuePairs() ([]interface{}, error)
}

type compoundObject struct {
	*object
	abstractCompoundObject
}

// The caller should check if return value is "".
func (o *compoundObject) genHashPrefix() string {
	if o.HashPrefix != "" {
		return o.HashPrefix
	}
	return o.typ.Name()
}

// The caller should check if return value is "".
func (o *compoundObject) genHashName() string {
	if o.HashName != "" {
		return o.HashName
	}

	ref := o.Reference
	if ref == "" || o.parent == nil {
		return ""
	}

	parent, ok := o.parent.abstractCompoundObject.(*structObject)
	if !ok {
		return ""
	}

	fld := parent.getFieldByName(ref)
	if fld == nil {
		return ""
	}

	po, ok := fld.abstractObject.(*plainObject)
	if !ok {
		return ""
	}

	return string(po.reply)
}

func (o *compoundObject) genRedisKey(ns string) (string, error) {
	key := o.genHashName()
	if key == "" {
		return "", newErrorObjectWithoutHashKey(o.name)
	}

	hashPrefix := o.genHashPrefix()
	redisKey := strings.Join([]string{ns, hashPrefix, key}, "#")
	return redisKey, nil
}

func (o *compoundObject) doRedisSave(conn redis.Conn, ns string) error {
	key, err := o.genRedisKey(ns)
	if err != nil {
		return err
	}

	cmdArgs, err := o.genHashFieldValuePairs()
	if err != nil {
		return err
	}

	args := []interface{}{key}
	args = append(args, cmdArgs...)
	_, err = conn.Do("HMSET", args...)
	if err != nil {
		return err
	}

	return nil
}

func newCompoundObject(o *object) (*compoundObject, error) {
	obj := &compoundObject{object: o}
	o.abstractObject = obj
	return obj, nil
}
