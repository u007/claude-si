//go:build windows

package filelock

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var (
	lockKernel32                     = syscall.NewLazyDLL("kernel32.dll")
	lockFileExProc                   = lockKernel32.NewProc("LockFileEx")
	unlockFileExProc                 = lockKernel32.NewProc("UnlockFileEx")
	errorLockViolation syscall.Errno = 33
)

func tryLockFile(file *os.File) error {
	var overlapped syscall.Overlapped
	ok, _, err := lockFileExProc.Call(uintptr(file.Fd()), uintptr(0x2|0x1), 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if ok == 0 {
		return windowsCallError(err)
	}
	return nil
}

func unlockFile(file *os.File) error {
	var overlapped syscall.Overlapped
	ok, _, err := unlockFileExProc.Call(uintptr(file.Fd()), 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if ok == 0 {
		return windowsCallError(err)
	}
	return nil
}

func isLockContention(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) && errno == errorLockViolation
}

func windowsCallError(err error) error {
	if err != nil {
		return err
	}
	return syscall.EINVAL
}
