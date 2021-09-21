package decommit

import (
	"os"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

var (
	ps  = uintptr(os.Getpagesize())
	psm uintptr
)

func init() {
	if ps&(ps-1) == 0 {
		psm = ps - 1
	}
}

// Slice attempts to decommit the memory that backs the provided slice,
// by asking the OS to decommit from memory as soon as possible the memory
// region that holds the slice contents.
// After the call, the slice contents are undetermined and may contain
// garbage.
// Slice affects the whole slice capacity, i.e. buf[0:cap(buf)].
// Slice returns how many bytes of memory were succesfully decommitted:
// while it attempts to decommit as much as possible, it entirely depends
// on whether and how the required functionality is exposed by the OS,
// and as such it may result in the slice not being decommited at all, or
// being decommitted only partially. Most operating systems place restrictions
// on the granularity of this function, so it normally is possible to only
// decommit whole memory pages (normally 4KB, but OS dependent: see
// os.Getpagesize()): Slice will automatically perform the required
// alignment operations, but this means that slices smaller than the page
// size will not be decommitted.
// Calling Slice passing as argument a slice that is not allocated by the Go
// runtime may result in unexpected side effects.
func Slice(buf []byte) int {
	buf = buf[:cap(buf)]
	if len(buf) < int(ps) || len(buf) == 0 {
		return 0
	}
	start := uintptr(unsafe.Pointer(&buf[0]))
	end := start + uintptr(len(buf))
	l := decommit(start, end)
	runtime.KeepAlive(buf)
	return l
}

// Any decommits the memory used by the provided obj.
// It returns the number of bytes that have been decommitted.
// Any relies on reflection to detect whether it is safe to decommit the
// memory used by obj: it is generally more performant to use Slice instead.
func Any(obj interface{}) int {
	start, length := findRegion(reflect.ValueOf(obj))
	if length < ps {
		return 0
	}
	l := decommit(start, start+length)
	runtime.KeepAlive(obj)
	return l
}

func findRegion(obj reflect.Value) (uintptr, uintptr) {
loop:
	switch obj.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !obj.IsNil() {
			obj = obj.Elem()
			goto loop
		}
	case reflect.Slice:
		if !containsPointers(obj.Type().Elem()) {
			return obj.Pointer(), uintptr(obj.Len() * int(obj.Type().Size()))
		}
	case reflect.Array:
		if !containsPointers(obj.Type().Elem()) && obj.CanAddr() {
			return obj.UnsafeAddr(), obj.Type().Size()
		}
	case reflect.Struct:
		if !containsPointers(obj.Type()) && obj.CanAddr() {
			return obj.UnsafeAddr(), obj.Type().Size()
		}
	case reflect.Map:
		// TODO: we may want to try to do something smart with the unused space
		// in the memory backing the map; for now we do nothing
	case reflect.Chan:
		// TODO: we may want to try to do something smart with the unused space
		// in the memory backing the channel; for now we do nothing
	case reflect.Func, reflect.String:
		// nothing to do
	case reflect.UnsafePointer:
		// we have no idea what the pointer is pointing to, so bail out
	default:
		// almost certainly not something we can decommit
	}
	return 0, 0
}

var typeMap sync.Map

func containsPointers(t reflect.Type) bool {
	if hasPtr, found := typeMap.Load(t); found {
		return hasPtr.(bool)
	}
	hasPtr := hasPointers(t)
	typeMap.Store(t, hasPtr)
	return hasPtr
}

func hasPointers(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Map, reflect.Chan, reflect.Ptr, reflect.Slice, reflect.Interface, reflect.Func, reflect.String, reflect.UnsafePointer:
		return true
	case reflect.Array:
		return hasPointers(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if hasPointers(t.Field(i).Type) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// Stack decommits the unused part of the calling goroutine stack.
func Stack() int {
	return 0
}

/*
import "constraints"

// noptr is a constraint for types that can not contain pointers or references
type noptr interface {
	// TODO: add array of noptr ([...]noptr)
	// TODO: add struct containing only noptr types
	constraints.Integer | constraints.Float | constraints.Complex | ~byte | ~rune | ~bool
}

func Slice[T noptr](s []T) int {
	buf = buf[:cap(buf)]
	if len(buf) < int(ps) || len(buf) == 0 {
		return 0
	}
	var zero T
	start := uintptr(unsafe.Pointer(&buf[0]))
	end := start + uintptr(len(buf)) * unsafe.Sizeof(zero)
	l := decommit(start, end)
	runtime.KeepAlive(buf)
	return l
}
*/

var decommitHook func(uintptr, uintptr, uintptr, uintptr, int) (uintptr, int) // for testing

func decommit(start, end uintptr) int {
	astart, aend, alength := pageAlign(start, end)
	if isTesting && decommitHook != nil {
		astart, alength = decommitHook(start, end, astart, aend, alength)
	}
	if alength == 0 {
		return 0
	}
	return osDecommit(astart, alength)
}

func pageAlign(start, end uintptr) (astart, aend uintptr, alength int) {
	if psm != 0 {
		astart = (start + psm) &^ psm
		aend = end &^ psm
	} else {
		astart = (start + ps - 1) / ps * ps
		aend = end / ps * ps
	}
	if astart >= aend {
		return 0, 0, 0
	}
	alength = int(aend - astart)
	return
}
