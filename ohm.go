package go_ohm

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/gomodule/redigo/redis"
)

type objectOptions struct {
	hashKey           string
	hashField         string
	reference         string
	shouldJsonify     bool
	shouldElemJsonify bool
}

type object struct {
	name   string
	fields []*object
	parent *object

	*objectOptions
	typ      reflect.Type
	value    *reflect.Value
	indirect int
	reply    []byte
}

var tagIdentifier = "go_ohm"

func isIgnoredType(typ reflect.Type) bool {
	knd := typ.Kind()
	return knd == reflect.Chan || knd == reflect.Func ||
		knd == reflect.Invalid || knd == reflect.UnsafePointer ||
		knd == reflect.Interface
}

func isCompoundType(typ reflect.Type) bool {
	knd := typ.Kind()
	return knd == reflect.Map || knd == reflect.Struct ||
		((knd == reflect.Slice || knd == reflect.Array) &&
			typ.Elem().Kind() != reflect.Uint8)
}

func objectConcreteType(typ reflect.Type,
	val *reflect.Value) (reflect.Type, *reflect.Value, int) {
	if val != nil && val.IsValid() {
		for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
			if val.IsZero() {
				break
			}

			v := val.Elem()
			val = &v
		}

		typ = val.Type()
	}

	indirect := 0
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		indirect++
	}

	return typ, val, indirect
}

func (o *object) isPlainObject() bool {
	return !isCompoundType(o.typ) || o.shouldJsonify
}

func (o *object) isTiledObject() bool {
	return o.typ.Kind() == reflect.Struct && o.name == "" && !o.shouldJsonify &&
		o.reference == "" && o.hashKey == ""
}

func (o *object) addField(obj *object) {
	o.fields = append(o.fields, obj)
}

func (o *object) getFields() []*object {
	return o.fields
}

func (o *object) getFieldByName(name string) *object {
	for _, f := range o.fields {
		if f.name == name {
			return f
		}
	}

	return nil
}

func (o *object) getPlainObjects() []*object {
	var ret []*object
	for _, fo := range o.fields {
		if fo.isTiledObject() {
			ret = append(ret, fo.getPlainObjects()...)
		} else if fo.isPlainObject() {
			ret = append(ret, fo)
		}
	}

	return ret
}

func (o *object) getForeignObjects() []*object {
	var ret []*object
	for _, fo := range o.fields {
		if fo.isTiledObject() {
			ret = append(ret, fo.getForeignObjects()...)
		} else if !fo.isPlainObject() {
			ret = append(ret, fo)
		}
	}

	return ret
}

func (o *object) getDescendants(objList *[]*object) {
	*objList = append(*objList, o)
	for _, frn := range o.getForeignObjects() {
		frn.getDescendants(objList)
	}
}

func (o *object) genHmgetHashField() string {
	if o.hashField != "" {
		return o.hashField
	}

	return o.name
}

func (o *object) genHmgetHashKey() string {
	if o.hashKey != "" {
		return o.hashKey
	}

	ref := o.reference
	if ref == "" || o.parent == nil {
		return ""
	}

	fld := o.parent.getFieldByName(ref)
	return string(fld.reply)
}

func (o *object) genHmgetArgs() []interface{} {
	var args []interface{}
	for _, po := range o.getPlainObjects() {
		args = append(args, po.genHmgetHashField())
	}

	return args
}

func (o *object) doRedisHmget(conn redis.Conn) error {
	key := o.genHmgetHashKey()
	if key == "" {
		return NewErrorObjectWithoutHashKey(o.name)
	}

	args := []interface{}{key}
	args = append(args, o.genHmgetArgs()...)

	rep, err := redis.ByteSlices(conn.Do("HMGET", args...))
	if err == redis.ErrNil {
		return nil
	} else if err != nil {
		_ = conn.Close()
		return NewErrorRedisCommandsFailed(err)
	}

	for i, po := range o.getPlainObjects() {
		po.reply = rep[i]
	}

	return nil
}

func (o *object) createValue() {
	if o.indirect <= 0 {
		return
	}

	v := o.value
	for i := 0; i < o.indirect; i++ {
		t := v.Type().Elem()
		p := reflect.New(t)
		v.Set(p)
		*v = p.Elem()
	}

	for _, fo := range o.getFields() {
		fv := o.value.FieldByName(fo.name)
		fo.value = &fv
	}
}

func renderArray(obj *object) {
}

func renderSlice(obj *object) {
}

func renderMap(obj *object) {
}

func renderStruct(obj *object) error {
	for _, o := range obj.getFields() {
		err := o.renderValue()
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *object) renderValue() error {
	if !o.isPlainObject() {
		o.createValue()

		switch o.typ.Kind() {
		case reflect.Struct:
			err := renderStruct(o)
			if err != nil {
				return err
			}
			break

		case reflect.Slice:
			renderSlice(o)
			break

		case reflect.Map:
			renderMap(o)
			break

		case reflect.Array:
			renderArray(o)
			break
		}
	} else {
		if o.reply == nil {
			return nil
		}

		o.createValue()

		if o.shouldJsonify {
			return json.Unmarshal(o.reply, o.value.Addr().Interface())
		}

		switch o.typ.Kind() {
		case reflect.String:
			o.value.SetString(string(o.reply))
			break

		case reflect.Int:
			fallthrough
		case reflect.Int8:
			fallthrough
		case reflect.Int16:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			n, _ := strconv.Atoi(string(o.reply))
			o.value.SetInt(int64(n))
			break

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
			n, _ := strconv.Atoi(string(o.reply))
			o.value.SetUint(uint64(n))
			break

		case reflect.Bool:
			n, _ := strconv.Atoi(string(o.reply))
			o.value.SetBool(n >= 1)
			break

		case reflect.Float32:
			fallthrough
		case reflect.Float64:
			n, _ := strconv.ParseFloat(string(o.reply), 64)
			o.value.SetFloat(n)
			break

		case reflect.Complex64:
			fallthrough
		case reflect.Complex128:
			n, _ := strconv.ParseComplex(string(o.reply), 128)
			o.value.SetComplex(n)
			break

		case reflect.Slice:
			if o.typ.Elem().Kind() == reflect.Uint8 {
				o.value.SetBytes(o.reply)
				break
			}
		}
	}
	return nil
}

func completeArray(obj *object) *object {
	return nil
}

func completeSlice(obj *object) *object {
	return nil
}

func completeMap(obj *object) *object {
	return nil
}

func parseObjectOptions(t string, opts *objectOptions) bool {
	if t == "" {
		return false
	} else if t == "-" {
		return true
	}

	processors := map[string]func(string){
		"hash_key": func(v string) {
			opts.hashKey = v
		},
		"hash_field": func(v string) {
			opts.hashField = v
		},
		"reference": func(v string) {
			opts.reference = v
		},
		"should_jsonify": func(v string) {
			opts.shouldJsonify = true
		},
		"should_elem_jsonify": func(v string) {
			opts.shouldElemJsonify = true
		},
	}

	parts := strings.Split(t, ",")
	for _, opt := range parts {
		opt := strings.TrimSpace(opt)
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

func completeStruct(obj *object) error {
	typ := obj.typ
	for i := 0; i < typ.NumField(); i++ {
		fld := typ.Field(i)
		if !fld.IsExported() {
			continue
		}

		fldOpts := &objectOptions{}
		ignore := parseObjectOptions(fld.Tag.Get(tagIdentifier), fldOpts)
		if ignore {
			continue
		}

		fldNam := ""
		if !fld.Anonymous {
			fldNam = fld.Name
		}

		fldTyp := fld.Type
		fldVal := (*reflect.Value)(nil)
		if obj.indirect <= 0 {
			fv := obj.value.Field(i)
			fldVal = &fv
		}

		fldObj, err := buildObject(fldNam, obj, fldOpts, fldTyp, fldVal)
		if err != nil {
			return err
		}

		obj.addField(fldObj)
	}

	return nil
}

func buildObject(name string, parent *object, opts *objectOptions,
	typ reflect.Type, val *reflect.Value) (*object, error) {
	objTyp, objVal, indirect := objectConcreteType(typ, val)
	if isIgnoredType(objTyp) {
		// do not support those types, skip.
		return nil, NewErrorUnsupportedObjectType(name)
	}

	obj := &object{
		name:          name,
		objectOptions: opts,
		typ:           objTyp,
		value:         objVal,
		indirect:      indirect,
		parent:        parent,
	}

	if obj.isPlainObject() {
		return obj, nil
	}

	if objTyp.Kind() == reflect.Struct {
		err := completeStruct(obj)
		if err != nil {
			return nil, err
		}
	} else if objTyp.Kind() == reflect.Array {
		completeArray(obj)
	} else if objTyp.Kind() == reflect.Slice {
		completeSlice(obj)
	} else if objTyp.Kind() == reflect.Map {
		completeMap(obj)
	}

	return obj, nil
}

func doLoadCommands(conn redis.Conn, obj *object) error {
	var objs []*object
	obj.getDescendants(&objs)

	for _, obj := range objs {
		err := obj.doRedisHmget(conn)
		if err != nil {
			return err
		}
	}

	return nil
}

func Load(conn redis.Conn, key string, data interface{}) error {
	typ := reflect.TypeOf(data)
	if typ == nil {
		return NewErrorUnsupportedObjectType("root object")
	}

	val := reflect.ValueOf(data)
	if !val.IsValid() {
		return NewErrorUnsupportedObjectType("root object")
	}

	objOpts := &objectOptions{
		hashKey:           key,
		hashField:         "",
		reference:         "",
		shouldJsonify:     false,
		shouldElemJsonify: false,
	}
	obj, err := buildObject("root object", nil, objOpts, typ, &val)
	if err != nil {
		return err
	}

	err = doLoadCommands(conn, obj)
	if err != nil {
		return err
	}

	return obj.renderValue()
}

func Save(key string, obj interface{}) error {
	return nil
}
