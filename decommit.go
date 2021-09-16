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

// Slice attempts to decommit the memory that backs the provided slice.
// After the call, the slice contents are undetermined and may contain
// garbage.
// Slice affects the whole slice capacity, i.e. buf[0:cap(buf)].
// Slice returns how many bytes of memory were succesfully decommitted:
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
	astart, aend, alength := pageAlign(start, end)
	if decommitHook != nil {
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
