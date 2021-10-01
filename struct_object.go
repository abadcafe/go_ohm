package go_ohm

import (
	"github.com/gomodule/redigo/redis"
	"reflect"
	"strings"
)

type structObject struct {
	*compoundObject

	// all exported fields of the struct, include anonymous struct.
	fields []*object
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
		if obj.isPromotedObject() {
			so := obj.abstractObject.(*compoundObject).abstractCompoundObject.(*structObject)
			ret = append(ret, so.getPlainFields()...)
		} else if obj.isPlainObject() {
			po := obj.abstractObject.(*plainObject)
			ret = append(ret, po)
		}
	}

	return ret
}

func (o *structObject) getForeignObjects() []*compoundObject {
	var ret []*compoundObject

	for _, obj := range o.fields {
		if obj.isPromotedObject() {
			so := obj.abstractObject.(*compoundObject).abstractCompoundObject.(*structObject)
			ret = append(ret, so.getForeignObjects()...)
		} else if !obj.isPlainObject() {
			ret = append(ret, obj.abstractObject.(*compoundObject))
		}
	}

	return ret
}

func (o *structObject) getDescendants(objList *[]*compoundObject) {
	*objList = append(*objList, o.compoundObject)
	for _, obj := range o.getForeignObjects() {
		obj.getDescendants(objList)
	}
}

func (o *structObject) genHashFields() []interface{} {
	var args []interface{}

	for _, obj := range o.getPlainFields() {
		args = append(args, obj.genHashField())
	}

	return args
}

func (o *structObject) genHashFieldValuePairs() ([]interface{}, error) {
	var args []interface{}

	for _, obj := range o.getPlainFields() {
		k := obj.genHashField()
		if k == "" {
			return nil, NewErrorUnsupportedObjectType(obj.name)
		}

		v, err := obj.genHashValue()
		if err != nil {
			return nil, NewErrorUnsupportedObjectType(obj.name)
		}

		args = append(args, k, v)
	}

	return args, nil
}

func parseObjectOptions(t string, opts *ObjectOptions) bool {
	if t == "" {
		return false
	} else if t == "-" {
		return true
	}

	processors := map[string]func(string){
		"hash_prefix": func(v string) {
			opts.hashPrefix = v
		},
		"hash_name": func(v string) {
			opts.hashName = v
		},
		"hash_field": func(v string) {
			opts.hashField = v
		},
		"reference": func(v string) {
			opts.reference = v
		},
		"json": func(v string) {
			opts.json = true
		},
		"non_json": func(v string) {
			opts.json = false
		},
		"elem_json": func(v string) {
			opts.elemNonJson = false
		},
		"elem_non_json": func(v string) {
			opts.elemNonJson = true
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

func (o *structObject) doRedisLoad(conn redis.Conn, ns string) error {
	key, err := o.genRedisKey(ns)
	if err != nil {
		return err
	}

	args := []interface{}{key}
	args = append(args, o.genHashFields()...)
	if len(args) <= 1 {
		// nothing to do.
		return nil
	}

	rep, err := redis.ByteSlices(conn.Do("HMGET", args...))
	if err != nil {
		return NewErrorRedisCommandFailed(o.name, err)
	}

	for i, po := range o.getPlainFields() {
		po.reply = rep[i]
	}

	return nil
}

func (o *structObject) renderValue() error {
	o.createIndirectValues()

	for _, fo := range o.getFields() {
		if o.indirect > 0 {
			fv := o.value.FieldByName(fo.name)
			fo.value = &fv
		}

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
		fldNam := fld.Name
		fldAnon := fld.Anonymous
		fldTyp := fld.Type
		fldVal := (*reflect.Value)(nil)

		if !fld.IsExported() {
			if !fldAnon {
				continue
			}

			// can not set unexported anonymous struct pointer field's value,
			// should skip those cases.
			if fldTyp.Kind() == reflect.Ptr {
				if o.indirect > 0 {
					// Case 1, parent is nil. Even allocated new struct, the
					// anonymous field is nil and can not be set.
					continue
				}

				v := o.value.Field(i)
				if v.IsNil() {
					// Case 2, the anonymous field itself is nil.
					continue
				}

				fldVal = &v
			}
		}

		if fldVal == nil && o.indirect <= 0 {
			v := o.value.Field(i)
			fldVal = &v
		}

		fldTyp, fldVal, indirect := advanceIndirectTypeAndValue(fldTyp, fldVal)
		if isIgnoredType(fldTyp) {
			// do not support those types, skip.
			return NewErrorUnsupportedObjectType(fldNam)
		}

		fldOpts := &ObjectOptions{}
		if !isPrimitiveType(fldTyp) && !fldAnon {
			// For primitive types, default to non json to improve performance,
			// And for anonymous fields, default to non json to promote its
			// fields.
			fldOpts.json = true
		}

		ignore := parseObjectOptions(fld.Tag.Get(tagIdentifier), fldOpts)
		if ignore {
			continue
		}

		fldObj, err := newObject(fldNam, o.compoundObject, fldOpts, fldTyp,
			fldVal, indirect, fldAnon)
		if err != nil {
			return err
		}

		o.addField(fldObj)
	}

	return nil
}

func newStructObject(co *compoundObject) (*structObject, error) {
	obj := &structObject{compoundObject: co}
	obj.abstractCompoundObject = obj
	err := obj.complete()
	if err != nil {
		return nil, err
	}

	return obj, nil
}
