package admin

import (
	"errors"
	"net/http"
	"reflect"
	"sync"
	"unsafe"
)

// unregisterHandlers 从 http.DefaultServeMux 中删除路由
// admin 包引入了 net/http/pprof，会自动在 http.DefaultServeMux 上注册 pprof 相关路由，引起安全问题
// 可参考：https://github.com/golang/go/issues/22085
func unregisterHandlers(patterns []string) error {
	// http.ServeMux 结构如下：
	// type ServeMux struct {
	// 	mu    sync.RWMutex
	// 	m     map[string]muxEntry
	// 	es    []muxEntry
	// 	hosts bool
	// }

	// 需要引入 net/http 包中的 muxEntry
	type muxEntry struct {
		h       http.Handler
		pattern string
	}

	v := reflect.ValueOf(http.DefaultServeMux)

	// 加锁
	muField := v.Elem().FieldByName("mu")
	if !muField.IsValid() {
		return errors.New("http.DefaultServeMux does not have a field called `mu`")
	}
	muPointer := unsafe.Pointer(muField.UnsafeAddr())
	mu := (*sync.RWMutex)(muPointer)
	(*mu).Lock()
	defer (*mu).Unlock()

	// 删除 map 中的值
	mField := v.Elem().FieldByName("m")
	if !mField.IsValid() {
		return errors.New("http.DefaultServeMux does not have a field called `m`")
	}
	mPointer := unsafe.Pointer(mField.UnsafeAddr())
	m := (*map[string]muxEntry)(mPointer)
	for _, pattern := range patterns {
		delete(*m, pattern)
	}

	// 删除 muxEntry slice 中的值
	esField := v.Elem().FieldByName("es")
	if !esField.IsValid() {
		return errors.New("http.DefaultServeMux does not have a field called `es`")
	}
	esPointer := unsafe.Pointer(esField.UnsafeAddr())
	es := (*[]muxEntry)(esPointer)
	for _, pattern := range patterns {
		// 删除相同 pattern 的 muxEntry
		var j int
		for _, muxEntry := range *es {
			if muxEntry.pattern != pattern {
				(*es)[j] = muxEntry
				j++
			}
		}
		*es = (*es)[:j]
	}

	// 修改 hosts
	hostsField := v.Elem().FieldByName("hosts")
	if !hostsField.IsValid() {
		return errors.New("http.DefaultServeMux does not have a field called `hosts`")
	}
	hostsPointer := unsafe.Pointer(hostsField.UnsafeAddr())
	hosts := (*bool)(hostsPointer)
	*hosts = false
	for _, v := range *m {
		if v.pattern[0] != '/' {
			*hosts = true
		}
	}

	return nil
}
