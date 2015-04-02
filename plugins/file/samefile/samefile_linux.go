// +build linux openbsd solaris dragonfly

package samefile

import (
	"encoding/binary"
	"io"
	"os"
	"syscall"
)

func fsHash(path string, h io.Writer) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	fstat := info.Sys().(*syscall.Stat_t)

	binary.Write(h, binary.BigEndian, fstat.Ino)
	binary.Write(h, binary.BigEndian, fstat.Dev)
	binary.Write(h, binary.BigEndian, fstat.Ctim.Sec)
	binary.Write(h, binary.BigEndian, fstat.Ctim.Nsec)

	return nil
}
