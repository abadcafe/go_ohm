package go_ohm

import (
	"encoding/json"
	"reflect"
)

func jsonMarshalValue(v *reflect.Value) ([]byte, error) {
	var i interface{}

	if v.Kind() == reflect.Ptr {
		i = v.Interface()
	} else {
		p := reflect.New(v.Type())
		p.Elem().Set(*v)
		i = p.Interface()
	}

	bs, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

func jsonUnmarshalValue(bs []byte, v *reflect.Value) error {
	var i interface{}

	if v.Kind() == reflect.Ptr {
		i = v.Interface()
	} else {
		i = v.Addr().Interface()
	}

	err := json.Unmarshal(bs, i)
	if err != nil {
		return err
	}

	return nil
}
