package go_ohm

import (
	"bytes"
	"errors"
	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
	"reflect"
	"testing"
)

func getTypeValue(i interface{}) (reflect.Type, *reflect.Value) {
	t := reflect.TypeOf(i)
	v := reflect.ValueOf(i)
	return t, &v
}

func TestAdvanceIndirectTypeAndValue(t *testing.T) {
	var d1 = 1
	var d2 = &d1
	var d3 = &d2
	var d4 interface{} = &d3
	var d5 interface{} = &d4
	var d6 ***int
	var d7 interface{} = &d6
	var d8 = &d7
	var d9 = &d8
	var d10 **interface{}
	var d11 = struct{ A interface{} }{}
	var d12 = struct{ A **interface{} }{d9}

	t.Run("test advanceIndirectTypeAndValue()", func(t *testing.T) {
		typ, val, ind := advanceIndirectTypeAndValue(getTypeValue(d1))
		if typ.Kind() != reflect.Int || val.IsZero() || ind != 0 {
			t.Error(typ, val, ind)
		}

		typ, val, ind = advanceIndirectTypeAndValue(getTypeValue(d2))
		if typ.Kind() != reflect.Int || val.IsZero() || ind != 0 {
			t.Error(typ, val, ind)
		}

		typ, val, ind = advanceIndirectTypeAndValue(getTypeValue(d3))
		if typ.Kind() != reflect.Int || val.IsZero() || ind != 0 {
			t.Error(typ, val, ind)
		}

		typ, val, ind = advanceIndirectTypeAndValue(getTypeValue(d4))
		if typ.Kind() != reflect.Int || val.IsZero() || ind != 0 {
			t.Error(typ, val, ind)
		}

		typ, val, ind = advanceIndirectTypeAndValue(getTypeValue(d5))
		if typ.Kind() != reflect.Int || val.IsZero() || ind != 0 {
			t.Error(typ, val, ind)
		}

		typ, val, ind = advanceIndirectTypeAndValue(getTypeValue(d6))
		if typ.Kind() != reflect.Int || !val.IsZero() || ind != 3 {
			t.Error(typ, val, ind)
		}

		typ, val, ind = advanceIndirectTypeAndValue(getTypeValue(d7))
		if typ.Kind() != reflect.Int || !val.IsZero() || ind != 3 {
			t.Error(typ, val, ind)
		}

		typ, val, ind = advanceIndirectTypeAndValue(getTypeValue(d8))
		if typ.Kind() != reflect.Int || !val.IsZero() || ind != 3 {
			t.Error(typ, val, ind)
		}

		typ, val, ind = advanceIndirectTypeAndValue(getTypeValue(d9))
		if typ.Kind() != reflect.Int || !val.IsZero() || ind != 3 {
			t.Error(typ, val, ind)
		}

		typ, val, ind = advanceIndirectTypeAndValue(getTypeValue(d10))
		if typ.Kind() != reflect.Interface || !val.IsZero() || ind != 2 {
			t.Error(typ, val, ind)
		}

		typ, val = getTypeValue(d11)
		typ = typ.Field(0).Type
		v := val.Field(0)
		val = &v
		typ, val, ind = advanceIndirectTypeAndValue(typ, val)
		if typ.Kind() != reflect.Interface || !val.IsZero() || ind != 0 {
			t.Error(typ, val, ind)
		}

		typ, val = getTypeValue(d12)
		typ = typ.Field(0).Type
		v = val.Field(0)
		val = &v
		typ, val, ind = advanceIndirectTypeAndValue(typ, val)
		if typ.Kind() != reflect.Int || !val.IsZero() || ind != 3 {
			t.Error(typ, val, ind)
		}
	})
}

func TestLoad(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	s.FlushAll()
	c, err := redis.Dial("tcp", s.Addr())

	type test3 struct {
		I3 int
	}

	type test2 struct {
		I int
	}

	type test1 struct {
		i      int       `go_ohm:"hash_field=i"` // ignored
		I2     uint      `go_ohm:"hash_field=i2"`
		F      **float32 `go_ohm:"hash_field=f"`
		S      **string  `go_ohm:"hash_field=s"`
		S2     **string  `go_ohm:"-"` // ignored
		S3     **string  `go_ohm:"hash_field=s3,json"`
		S4     **test2   `go_ohm:"hash_field=s4"`
		S5     **test2   `go_ohm:"hash_field=s5,hash_key=test2,non_json"`
		*test2           // ignored
		test3  `go_ohm:"hash_key=test3"`
		B      bool            `go_ohm:"hash_field=b"`
		B2     []byte          `go_ohm:"hash_field=b2"`
		M      *map[string]int `go_ohm:"hash_prefix=test4,hash_key=test4,non_json"`
	}

	t.Run("test Load() unsupported types", func(t *testing.T) {
		var e *ErrorUnsupportedObjectType

		var v1 interface{}
		err := Load(c, "test", &ObjectOptions{hashKey: "test1"}, v1)
		if !errors.As(err, &e) {
			t.Error(err)
		}

		var v2 chan int
		err = Load(c, "test", &ObjectOptions{hashKey: "test1"}, v2)
		if !errors.As(err, &e) {
			t.Error(err)
		}

		var v3 chan int
		err = Load(c, "test", &ObjectOptions{hashKey: "test1"}, &v3)
		if !errors.As(err, &e) {
			t.Error(err)
		}

		v4 := struct{ A **interface{} }{}
		err = Load(c, "test", &ObjectOptions{hashKey: "test1"}, &v4)
		if !errors.As(err, &e) {
			t.Error(err)
		}

		var e2 *ErrorObjectWithoutHashKey
		v5 := struct{ A **int }{}
		err = Load(c, "test", &ObjectOptions{}, &v5)
		if !errors.As(err, &e2) {
			t.Error(err)
		}
	})

	t.Run("test Load() nil", func(t *testing.T) {
		t1 := &test1{}
		err := Load(c, "test", &ObjectOptions{hashKey: "test1"}, t1)
		if err != nil {
			t.Error(err)
		} else if t1.i != 0 || t1.I2 != 0 || t1.F != nil || t1.S != nil ||
			t1.S2 != nil || t1.test2 != nil || t1.B != false || t1.B2 != nil {
			t.Error("wrong value: ", t1)
		}
	})

	t.Run("test Load() normal", func(t *testing.T) {
		s.HSet("test#test4#test4", "ss", "2")
		s.HSet("test#test3#test2", "I3", "2")
		s.HSet("test#test2#test2", "I", "2")
		s.HSet(
			"test#test1#test1",
			"i", "2",
			"i2", "2",
			"f", "2.0",
			"s", "2",
			"s2", "2",
			"s3", "\"2\"",
			"s4", "{\"I\": 2}",
			"c", "2",
			"b", "2",
			"b2", "2",
		)
		t1 := &test1{}
		err = Load(c, "test", &ObjectOptions{hashKey: "test1"}, t1)
		if err != nil {
			t.Error(err)
		} else if t1.i != 0 || t1.I2 != 2 || **t1.F != 2.0 || **t1.S != "2" ||
			t1.S2 != nil || **t1.S3 != "2" || (**t1.S4).I != 2 ||
			(**t1.S5).I != 2 || t1.I3 == 2 || t1.B != true ||
			!bytes.Equal(t1.B2, []byte("2")) || (*t1.M)["ss"] == 2 {
			t.Errorf("wrong value: %++v, %v", t1, t1.S3)
		}
	})
}
