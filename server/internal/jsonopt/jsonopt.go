// Package jsonopt has a JSON field type that tells "absent" from "null"
// from a value, which a plain pointer can't. PATCH-style requests use it:
// absent leaves a field alone, null clears it, a value sets it.
package jsonopt

import "encoding/json"

// Optional is one such field. Set is true when the key was present in the
// JSON, and Value is nil when it was present as null.
type Optional[T any] struct {
	Set   bool
	Value *T
}

func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}
