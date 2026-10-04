package json

import (
	"fmt"
	"reflect"
)

// The helpers below fold a database.Value into a decoded row. Because the whole
// row is in memory here, union and difference are ordinary slice and map edits
// rather than the delete-and-reinsert the SQL backends need.

// copyValue returns an addressable shallow copy of a slice or map, so an
// in-place edit never mutates the value the caller handed us. Addressability
// matters because MakeSlice and MakeMapWithSize both return unsettable values.
func copyValue(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Slice:
		out := reflect.New(v.Type()).Elem()
		out.Set(reflect.MakeSlice(v.Type(), v.Len(), v.Len()))
		reflect.Copy(out, v)
		return out
	case reflect.Map:
		out := reflect.New(v.Type()).Elem()
		out.Set(reflect.MakeMapWithSize(v.Type(), v.Len()))
		iter := v.MapRange()
		for iter.Next() {
			out.SetMapIndex(iter.Key(), iter.Value())
		}
		return out
	default:
		return v
	}
}

// mergeValues adds the elements or entries of src into dst, skipping duplicates.
// A nil or empty src leaves dst untouched.
func mergeValues(dst, src reflect.Value) error {
	if !src.IsValid() {
		return nil
	}
	if dst.Kind() == reflect.Map {
		return mergeMap(dst, src)
	}
	if dst.Kind() != reflect.Slice && dst.Kind() != reflect.Array {
		return fmt.Errorf("json: cannot merge %s into %s", src.Kind(), dst.Kind())
	}
	if src.Kind() != reflect.Slice && src.Kind() != reflect.Array {
		return fmt.Errorf("json: cannot merge %s into a slice", src.Kind())
	}

	out := reflect.MakeSlice(dst.Type(), 0, dst.Len()+src.Len())
	seen := make(map[string]bool, dst.Len()+src.Len())
	appendAll := func(v reflect.Value) {
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i)
			id := stringOf(item)
			if seen[id] {
				continue
			}
			seen[id] = true
			out = reflect.Append(out, item)
		}
	}
	appendAll(dst)
	appendAll(src)

	dst.Set(out)
	return nil
}

func mergeMap(dst, src reflect.Value) error {
	if src.Kind() != reflect.Map {
		return fmt.Errorf("json: cannot merge %s into a map", src.Kind())
	}
	out := copyValue(dst)
	iter := src.MapRange()
	for iter.Next() {
		out.SetMapIndex(iter.Key(), iter.Value())
	}
	dst.Set(out)
	return nil
}

// removeValues deletes from dst everything src lists. dst must be a copy, since
// the caller mutates it in place.
func removeValues(dst, src reflect.Value) error {
	if !src.IsValid() {
		return nil
	}
	if dst.Kind() == reflect.Map {
		return removeMapEntries(dst, src)
	}
	if dst.Kind() != reflect.Slice && dst.Kind() != reflect.Array {
		return fmt.Errorf("json: cannot remove %s from %s", src.Kind(), dst.Kind())
	}

	drop := make(map[string]bool)
	if src.Kind() == reflect.Slice || src.Kind() == reflect.Array {
		for i := 0; i < src.Len(); i++ {
			drop[stringOf(src.Index(i))] = true
		}
	}

	out := reflect.MakeSlice(dst.Type(), 0, dst.Len())
	for i := 0; i < dst.Len(); i++ {
		if drop[stringOf(dst.Index(i))] {
			continue
		}
		out = reflect.Append(out, dst.Index(i))
	}
	dst.Set(out)
	return nil
}

func removeMapEntries(dst, src reflect.Value) error {
	if src.Kind() != reflect.Map {
		return fmt.Errorf("json: cannot remove %s from a map", src.Kind())
	}
	out := copyValue(dst)
	iter := src.MapRange()
	for iter.Next() {
		out.SetMapIndex(iter.Key(), reflect.Value{})
	}
	dst.Set(out)
	return nil
}
