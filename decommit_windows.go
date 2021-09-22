//go:build windows
// +build windows

package decommit

import (
	"reflect"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32DLL                = windows.NewLazySystemDLL("kernel32.dll")
	procDiscardVirtualMemory   = kernel32DLL.NewProc("DiscardVirtualMemory")
	procDiscardVirtualMemoryOK = procDiscardVirtualMemory.Find() == nil
)

func osDecommit(astart uintptr, alength int) int {
	if !procDiscardVirtualMemoryOK {
		return 0
	}
	ret, _, err := procDiscardVirtualMemory.Call(astart, alength)
	if err != nil || ret != 0 {
		return 0
	}
	return alength
}
