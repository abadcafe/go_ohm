package go_ohm

import (
	"encoding/json"
	"reflect"
	"strconv"
)

type plainObject struct {
	*object

	// redis reply of a redis hash field.
	reply []byte
}

func (o *plainObject) genHashField() string {
	if o.hashField != "" {
		return o.hashField
	}

	return o.name
}

func (o *plainObject) renderValue() error {
	if o.reply == nil || len(o.reply) <= 0 {
		return nil
	}

	o.createIndirectValues()

	if !o.nonJson {
		err := json.Unmarshal(o.reply, o.value.Addr().Interface())
		if err != nil {
			return NewErrorJsonFailed(o.name, err)
		}

		return nil
	}

	switch o.typ.Kind() {
	case reflect.String:
		o.value.SetString(string(o.reply))

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

	case reflect.Bool:
		n, _ := strconv.Atoi(string(o.reply))
		o.value.SetBool(n >= 1)

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		n, _ := strconv.ParseFloat(string(o.reply), 64)
		o.value.SetFloat(n)

	case reflect.Complex64:
		fallthrough
	case reflect.Complex128:
		n, _ := strconv.ParseComplex(string(o.reply), 128)
		o.value.SetComplex(n)

	case reflect.Slice:
		if o.typ.Elem().Kind() == reflect.Uint8 {
			o.value.SetBytes(o.reply)
		}
	}

	return nil
}

func newPlainObject(o *object) (*plainObject, error) {
	obj := &plainObject{object: o}
	o.abstractObject = obj
	return obj, nil
}
