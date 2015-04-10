// +build linux

package samefile

import (
	"encoding/binary"
	"io"
	"os"
	"syscall"
	"time"
)

const cBirthTime = "user.cypress_btime"

func fsHash(path string, h io.Writer) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	fstat := info.Sys().(*syscall.Stat_t)

	binary.Write(h, binary.BigEndian, fstat.Ino)
	binary.Write(h, binary.BigEndian, fstat.Dev)

	dest := make([]byte, 64)

	n, err := syscall.Getxattr(path, cBirthTime, dest)
	if err == nil {
		h.Write(dest[:n])
		return nil
	}

	bin, err := time.Now().MarshalBinary()
	if err == nil {
		err = syscall.Setxattr(path, cBirthTime, bin, 0)
		if err == nil {
			h.Write(bin)
		}
	}

	return nil
}
