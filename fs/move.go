package fs

import (
	"io"
	"os"
	"time"
)

func Move(oldpath, newpath string) error {
	//from, err := windows.UTF16PtrFromString(oldpath)
	//if err != nil {
	//	return err
	//}
	//to, err := win.UTF16PtrFromString(newpath)
	//if err != nil {
	//	return err
	//}
	//return win.MoveFileEx(from, to, MOVEFILE_COPY_ALLOWED)
	return nil
}

func Copy(oldpath, newpath string) error {
	info, err := os.Stat(oldpath)
	if err != nil {
		return err
	}

	src, err := os.Open(oldpath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(newpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return os.Chtimes(newpath, time.Now(), info.ModTime())
}
