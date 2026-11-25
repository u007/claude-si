//go:build windows

package filelock

import (
	"syscall"
	"unsafe"
)

var (
	replaceKernel32 = syscall.NewLazyDLL("kernel32.dll")
	moveFileExProc  = replaceKernel32.NewProc("MoveFileExW")
)

func replaceFile(oldPath, newPath string) error {
	oldPtr, err := syscall.UTF16PtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPtr, err := syscall.UTF16PtrFromString(newPath)
	if err != nil {
		return err
	}
	ok, _, callErr := moveFileExProc.Call(uintptr(unsafe.Pointer(oldPtr)), uintptr(unsafe.Pointer(newPtr)), uintptr(0x1|0x8))
	if ok == 0 {
		return windowsCallError(callErr)
	}
	return nil
}
