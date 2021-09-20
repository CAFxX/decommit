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
