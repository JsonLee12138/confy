package confy

import (
	"reflect"

	"github.com/JsonLee12138/confy/core"
)

// These aliases preserve root-package usage. Canonical contracts live in core.
type Provider = core.Provider
type WatchProvider = core.WatchProvider
type Watcher = core.Watcher
type Change = core.Change

func isNilProvider(p Provider) bool {
	if p == nil {
		return true
	}
	v := reflect.ValueOf(p)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Func, reflect.Interface, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}
