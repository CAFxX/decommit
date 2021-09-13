package decommit

import (
	"os"
	"runtime"
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

/*
func Slice[T any](s []T) int {
    s = s[:cap(s)]
    var zero T
    if len(s) * unsafe.Sizeof(zero) < ps {
        return 0
    }
    start := uintptr(unsafe.Pointer(&s[0]))
    end := start + len(s) * unsafe.Sizeof(zero)
	l := decommit(start, end)
	runtime.KeepAlive(s)
	return l
}

func Range(ptr unsafe.Pointer, size uintptr) int {
	if size < ps || ptr == nil {
		return 0
	}
	start := uintptr(ptr)
	end := start + size
	l := decommit(start, end)
	runtime.KeepAlive(ptr)
	return l
}
*/

// Slice attempts to decommit the memory that backs the provided slice.
// After the call, the slice contents are undetermined and may contain
// garbage.
// It returns how many bytes of memory were succesfully decommitted:
// while it attempts to decommit as much as possible, it entirely depends
// on whether and how the required functionality is exposed by the OS,
// and as such it may result in the slice not being decommited at all, or
// being decommitted only partially.
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

var decommitHook func(uintptr, uintptr, uintptr, uintptr, int) (uintptr, int) // for testing

func decommit(start, end uintptr) int {
	var astart, aend uintptr
	if psm != 0 {
		astart = (start + psm) &^ psm
		aend = end &^ psm
	} else {
		astart = (start + ps - 1) / ps * ps
		aend = end / ps * ps
	}
	alength := int(aend - astart)
	if decommitHook != nil {
		astart, alength = decommitHook(start, end, astart, aend, alength)
	}
	if alength == 0 {
		return 0
	}
	return osDecommit(astart, alength)
}
