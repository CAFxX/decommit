//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris) && gc && !ppc64le && !ppc64
// +build darwin dragonfly freebsd linux netbsd openbsd solaris
// +build gc
// +build !ppc64le
// +build !ppc64

package decommit

import (
	"reflect"
	"unsafe"

	"golang.org/x/sys/unix"
)

func osDecommit(astart uintptr, alength int) int {
	var mem []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&mem))
	sh.Data = astart
	sh.Len = alength
	sh.Cap = alength
	if unix.Madvise(mem, unix.MADV_DONTNEED) != nil {
		return 0
	}
	return alength
}
