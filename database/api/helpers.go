package api

import "reflect"

const KindStruct = reflect.Struct

func UnwrapValueOf(v any) reflect.Value {
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val
}

func TypeOf(v any) reflect.Type {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func NewStruct(typ reflect.Type) reflect.Value {
	return reflect.New(typ).Elem()
}
