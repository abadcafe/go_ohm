package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
	"strings"
)

type structObject struct {
	*object

	// all exported fields of the struct, include anonymous struct fields.
	fields   []*object
}

func (o *structObject) addField(obj *object) {
	o.fields = append(o.fields, obj)
}

func (o *structObject) getFields() []*object {
	return o.fields
}

func (o *structObject) getFieldByName(name string) *object {
	for _, f := range o.fields {
		if f.name == name {
			return f
		}
	}

	return nil
}

func (o *structObject) getPlainFields() []*plainObject {
	var ret []*plainObject

	for _, obj := range o.fields {
		if obj.isTiledObject() {
			so := obj.concreteObject.(*structObject)
			ret = append(ret, so.getPlainFields()...)
		} else if obj.isPlainObject() {
			po := obj.concreteObject.(*plainObject)
			ret = append(ret, po)
		}
	}

	return ret
}

func (o *structObject) getForeignObjects() []*object {
	var ret []*object
	for _, obj := range o.fields {
		if obj.isTiledObject() {
			so := obj.concreteObject.(*structObject)
			ret = append(ret, so.getForeignObjects()...)
		} else if !obj.isPlainObject() {
			ret = append(ret, obj)
		}
	}

	return ret
}

func (o *structObject) getDescendants(objList *[]*object) {
	*objList = append(*objList, o.object)
	for _, obj := range o.getForeignObjects() {
		co := obj.concreteObject.(compoundObject)
		co.getDescendants(objList)
	}
}

// The caller should check if return value is "".
func (o *structObject) genHMGetHashKey() string {
	if o.hashKey != "" {
		return o.hashKey
	}

	ref := o.reference
	if ref == "" || o.parent == nil {
		return ""
	}

	parent := o.parent.concreteObject.(*structObject)
	fld := parent.getFieldByName(ref)
	v := fld.concreteObject.(*plainObject).reply
	return string(v)
}

// The caller should check if return value is "".
func (o *structObject) genHMGetHashPrefix() string {
	if o.hashPrefix != "" {
		return o.hashPrefix
	}

	return o.typ.Name() + "#"
}

func (o *structObject) genHMGetArgs() []interface{} {
	var args []interface{}
	for _, obj := range o.getPlainFields() {
		args = append(args, obj.genHMGetHashField())
	}

	return args
}

func parseObjectOptions(t string, opts *objectOptions) bool {
	if t == "" {
		return false
	} else if t == "-" {
		return true
	}

	processors := map[string]func(string){
		"hash_prefix": func(v string) {
			opts.hashPrefix = v
		},
		"hash_key": func(v string) {
			opts.hashKey = v
		},
		"hash_field": func(v string) {
			opts.hashField = v
		},
		"reference": func(v string) {
			opts.reference = v
		},
		"non_json": func(v string) {
			opts.nonJson = true
		},
		"elem_non_json": func(v string) {
			opts.elemNonJson = true
		},
		"json": func(v string) {
			opts.nonJson = false
		},
		"elem_json": func(v string) {
			opts.elemNonJson = false
		},
	}

	parts := strings.Split(t, ",")
	for _, opt := range parts {
		opt = strings.TrimSpace(opt)
		pair := strings.Split(opt, "=")
		if proc, ok := processors[strings.TrimSpace(pair[0])]; ok {
			arg := ""
			if len(pair) >= 2 {
				arg = pair[1]
			}

			proc(strings.TrimSpace(arg))
		}
	}

	return false
}

func (o *structObject) doRedisHMGet(conn redis.Conn, prefix string) error {
	key := o.genHMGetHashKey()
	if key == "" {
		return NewErrorObjectWithoutHashKey(o.name)
	}

	hashPrefix := o.genHMGetHashPrefix()
	key = prefix + hashPrefix + key

	args := []interface{}{key}
	args = append(args, o.genHMGetArgs()...)

	rep, err := redis.ByteSlices(conn.Do("HMGET", args...))
	if err != nil {
		return NewErrorRedisCommandsFailed(o.name, err)
	}

	for i, po := range o.getPlainFields() {
		po.reply = rep[i]
	}

	return nil
}

func (o *structObject) renderValue() error {
	o.createIndirectValues()

	for _, fo := range o.getFields() {
		fv := o.value.FieldByName(fo.name)
		fo.value = &fv
	}

	for _, fo := range o.getFields() {
		err := fo.renderValue()
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *structObject) complete() error {
	typ := o.typ
	for i := 0; i < typ.NumField(); i++ {
		fld := typ.Field(i)
		if !fld.IsExported() {
			continue
		}

		fldTyp := fld.Type
		fldVal := (*reflect.Value)(nil)
		if o.indirect <= 0 {
			fv := o.value.Field(i)
			fldVal = &fv
		}

		fldNam := ""
		if !fld.Anonymous {
			fldNam = fld.Name
		}

		fldTyp, fldVal, indirect := objectConcreteType(fldTyp, fldVal)
		if isIgnoredType(fldTyp) {
			// do not support those types, skip.
			return NewErrorUnsupportedObjectType(fldNam)
		}

		fldOpts := &objectOptions{}
		if isPrimitiveType(fldTyp) {
			// for primitive types, default to non json to improve performance.
			fldOpts.nonJson = true
		}

		ignore := parseObjectOptions(fld.Tag.Get(tagIdentifier), fldOpts)
		if ignore {
			continue
		}

		fldObj, err := buildObject(fldNam, o.object, fldOpts, fldTyp, fldVal,
			indirect)
		if err != nil {
			return err
		}

		o.addField(fldObj)
	}

	return nil
}

func completeStructObject(bo *object) error {
	obj := &structObject{object: bo}
	bo.concreteObject = obj
	return obj.complete()
}
