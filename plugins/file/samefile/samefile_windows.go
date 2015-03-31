// +build windows
import (
	"encoding/binary"
	"io"
	"syscall"
)

// shamelessly cribbed from https://golang.org/src/os/types_windows.go
func fsHash(path string, h io.Writer) error {
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}

	h, err := syscall.CreateFile(pathp, 0, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return err
	}

	defer syscall.CloseHandle(h)
	var i syscall.ByHandleFileInformation

	err = syscall.GetFileInformationByHandle(syscall.Handle(h), &i)
	if err != nil {
		return err
	}

	binary.Write(h, binary.BigEndian, i.VolumeSerialNumber)
	binary.Write(h, binary.BigEndian, i.FileIndexHigh)
	binary.Write(h, binary.BigEndian, i.FileIndexLow)

	return nil
}
