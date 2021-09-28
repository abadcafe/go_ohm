package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"strings"
)

type abstractCompoundObject interface {
	abstractObject
	getDescendants(objList *[]*compoundObject)
	doRedisHMGet(conn redis.Conn, ns string) error
}

type compoundObject struct {
	*object
	abstractCompoundObject
}

// The caller should check if return value is "".
func (o *compoundObject) genHashPrefix() string {
	if o.hashPrefix != "" {
		return o.hashPrefix
	}

	return o.typ.Name()
}

// The caller should check if return value is "".
func (o *compoundObject) genHashKey() string {
	if o.hashKey != "" {
		return o.hashKey
	}

	ref := o.reference
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

func (o *compoundObject) genRedisHashKey(prefix string) (string, error) {
	key := o.genHashKey()
	if key == "" {
		return "", NewErrorObjectWithoutHashKey(o.name)
	}

	hashPrefix := o.genHashPrefix()
	key = strings.Join([]string{prefix, hashPrefix, key}, "#")
	return key, nil
}

func newCompoundObject(o *object) (*compoundObject, error) {
	obj := &compoundObject{object: o}
	o.abstractObject = obj
	return obj, nil
}
