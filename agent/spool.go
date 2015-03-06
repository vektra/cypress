package agent

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/gogo/protobuf/proto"
	"github.com/vektra/cypress"
	"github.com/vektra/tai64n"
)

type SpoolFile struct {
	root    string
	current string
	file    *os.File
	bytes   int64

	feeder chan *cypress.Message
}

const PerFileSize = (1024 * 1024) // 1 meg per file
const MaxFiles = 10

// So we'll spool 10 megs worth of logs max

const DefaultSpoolDir = "/var/lib/vk-log/spool"

func (sf *SpoolFile) openCurrent() error {
	fd, err := os.OpenFile(sf.current, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	sf.file = fd

	fi, err := fd.Stat()

	if err != nil {
		return err
	}

	sf.bytes = fi.Size()

	return nil
}

func (sf *SpoolFile) Start(root string) error {
	sf.root = root

	sf.feeder = make(chan *cypress.Message)

	sf.pruneOldFiles()

	sf.current = path.Join(root, "current")

	err := sf.openCurrent()

	if err != nil {
		return err
	}

	go sf.process()

	return nil
}

func (sf *SpoolFile) newFilename() string {
	return path.Join(sf.root, tai64n.Now().Label())
}

func (sf *SpoolFile) pruneOldFiles() {
	files, err := ioutil.ReadDir(sf.root)

	if err != nil {
		fmt.Printf("Error reading files in %s: %s", sf.root, err)
		return
	}

	var oldest string = ""
	var oldest_ts *tai64n.TAI64N

	count := 0

	for _, fi := range files {
		ts := tai64n.ParseTAI64NLabel(fi.Name())

		if ts == nil {
			continue
		}

		count++

		if oldest_ts == nil || ts.Before(oldest_ts) {
			oldest_ts = ts
			oldest = fi.Name()
		}
	}

	if count <= MaxFiles {
		return
	}

	if oldest != "" {
		name := path.Join(sf.root, oldest)

		err := os.Remove(name)

		if err != nil {
			fmt.Printf("Error removing %s: %s\n", name, err)
		}
	}
}

func (sf *SpoolFile) process() {
	for {
		m := <-sf.feeder

		data, err := proto.Marshal(m)

		if err == nil {
			var buf [4]byte

			binary.BigEndian.PutUint32(buf[:], uint32(len(data)))

			_, err = sf.file.Write(buf[:])

			if err != nil {
				fmt.Printf("Error writing to current: %s\n", err)
			}

			_, err = sf.file.Write(data)

			if err != nil {
				fmt.Printf("Error writing to current: %s\n", err)
			}

			sf.file.Sync()

			sf.bytes += int64(len(data) + 4)

			if sf.bytes >= PerFileSize {
				sf.file.Close()
				os.Rename(sf.current, sf.newFilename())

				sf.pruneOldFiles()

				err = sf.openCurrent()

				if err != nil {
					fmt.Printf("Unable to open new current: %s", err)
					return
				}
			}
		}
	}
}

func (sf *SpoolFile) Read(m *cypress.Message) error {
	sf.feeder <- m
	return nil
}
