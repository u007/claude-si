//go:build windows

package filelock

import (
	"errors"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	errorAccessDenied     syscall.Errno = 5
	errorSharingViolation syscall.Errno = 32
)

var (
	replaceKernel32 = syscall.NewLazyDLL("kernel32.dll")
	moveFileExProc  = replaceKernel32.NewProc("MoveFileExW")
)

// replaceFile retries only when an editor or antivirus briefly holds the target.
// Any other failure is returned on the first attempt.
func replaceFile(oldPath, newPath string) error {
	return retryReplace(newPath, os.Stderr, isReplaceContention, time.Sleep, func() error {
		return moveFileEx(oldPath, newPath)
	})
}

func isReplaceContention(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) && (errno == errorAccessDenied || errno == errorSharingViolation)
}

func moveFileEx(oldPath, newPath string) error {
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
