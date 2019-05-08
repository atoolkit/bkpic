package windows

import (
	"syscall"
	"unsafe"
)

const (
	errnoERROR_IO_PENDING                 = 997
	COPY_FILE_FAIL_IF_EXISTS              = 0x00000001
	COPY_FILE_ALLOW_DECRYPTED_DESTINATION = 0x00000008
	MOVEFILE_COPY_ALLOWED                 = 0x2
)

var (
	modkernel32     = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileExW = modkernel32.NewProc("MoveFileExW")
	procCopyFileExW = modkernel32.NewProc("CopyFileExW")

	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
)

func Move(oldpath, newpath string) error {
	from, err := syscall.UTF16PtrFromString(oldpath)
	if err != nil {
		return err
	}
	to, err := syscall.UTF16PtrFromString(newpath)
	if err != nil {
		return err
	}
	return moveFileEx(from, to, MOVEFILE_COPY_ALLOWED)
}

func Copy(oldpath, newpath string) error {
	from, err := syscall.UTF16PtrFromString(oldpath)
	if err != nil {
		return err
	}
	to, err := syscall.UTF16PtrFromString(newpath)
	if err != nil {
		return err
	}
	return copyFileEx(from, to, COPY_FILE_FAIL_IF_EXISTS|COPY_FILE_ALLOW_DECRYPTED_DESTINATION)
}

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}

func copyFileEx(from *uint16, to *uint16, flags uint32) (err error) {
	r1, _, e1 := syscall.Syscall6(procCopyFileExW.Addr(), 6, uintptr(unsafe.Pointer(from)), uintptr(unsafe.Pointer(to)), 0, 0, 0, uintptr(flags))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
func moveFileEx(from *uint16, to *uint16, flags uint32) (err error) {
	r1, _, e1 := syscall.Syscall(procMoveFileExW.Addr(), 3, uintptr(unsafe.Pointer(from)), uintptr(unsafe.Pointer(to)), uintptr(flags))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
