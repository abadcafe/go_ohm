package go_ohm

import (
	"fmt"
	"reflect"
	"strconv"
)

type plainObject struct {
	*object

	// redis reply of a redis hash field.
	reply []byte
}

func (o *plainObject) genHashField() string {
	if o.HashField != "" {
		return o.HashField
	}
	return o.name
}

func (o *plainObject) genHashValue() (string, error) {
	if o.value == nil || !o.value.IsValid() || o.indirect > 0 {
		return "", nil
	}

	if o.Json {
		bs, err := jsonMarshalValue(o.value)
		if err != nil {
			return "", newErrorJsonFailed(o.name, err)
		}

		return string(bs), nil
	}

	if o.typ.Kind() == reflect.Slice && o.typ.Elem().Kind() == reflect.Uint8 {
		// byte slice.
		return string(o.value.Bytes()), nil
	}

	return fmt.Sprint(o.value.Interface()), nil
}

func (o *plainObject) renderValue() error {
	if o.reply == nil || len(o.reply) <= 0 {
		return nil
	}

	o.createIndirectValues()

	if o.Json {
		err := jsonUnmarshalValue(o.reply, o.value)
		if err != nil {
			return newErrorJsonFailed(o.name, err)
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
		i, err := strconv.Atoi(string(o.reply))
		if err != nil {
			return newErrorUnsupportedObjectType(o.name)
		}
		o.value.SetInt(int64(i))

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
		u, err := strconv.Atoi(string(o.reply))
		if err != nil {
			return newErrorUnsupportedObjectType(o.name)
		}
		o.value.SetUint(uint64(u))

	case reflect.Bool:
		b, err := strconv.ParseBool(string(o.reply))
		if err != nil {
			return newErrorUnsupportedObjectType(o.name)
		}
		o.value.SetBool(b)

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		f, err := strconv.ParseFloat(string(o.reply), 64)
		if err != nil {
			return newErrorUnsupportedObjectType(o.name)
		}
		o.value.SetFloat(f)

	case reflect.Complex64:
		fallthrough
	case reflect.Complex128:
		c, err := strconv.ParseComplex(string(o.reply), 128)
		if err != nil {
			return newErrorUnsupportedObjectType(o.name)
		}
		o.value.SetComplex(c)

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
