package plugin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"

	"code.google.com/p/snappy-go/snappy"

	"github.com/vektra/cypress"
	"github.com/vektra/tai64n"
)

type Spool struct {

	// The size of each file will get before it's rotated
	PerFileSize int64

	// How many rotate files to keep
	MaxFiles int

	root    string
	current string
	file    *os.File
	output  io.Writer
	bytes   int64

	enc *cypress.Encoder
}

const PerFileSize = (1024 * 1024) // 1 meg per file
const MaxFiles = 10

// So we'll spool 10 megs worth of logs max

const DefaultSpoolDir = "/var/lib/cypress/spool"

func (sf *Spool) openCurrent() error {
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

	if sf.bytes == 0 {
		hdr := &cypress.StreamHeader{
			Compression: cypress.StreamHeader_SNAPPY.Enum(),
		}

		var buf bytes.Buffer
		buf.WriteString("-")

		data, err := hdr.Marshal()
		if err != nil {
			return err
		}

		szbuf := make([]byte, 10)

		cnt := binary.PutUvarint(szbuf, uint64(len(data)))

		buf.Write(szbuf[:cnt])
		buf.Write(data)

		sf.file.Write(buf.Bytes())
	}

	sf.output = snappy.NewWriter(sf.file)

	sf.enc = cypress.NewEncoder(sf.output)

	return nil
}

func NewSpool(root string) (*Spool, error) {
	sf := &Spool{
		PerFileSize: PerFileSize,
		MaxFiles:    MaxFiles,
	}

	sf.root = root

	sf.pruneOldFiles()

	sf.current = path.Join(root, "current")

	err := sf.openCurrent()

	if err != nil {
		return nil, err
	}

	return sf, nil
}

func (sf *Spool) newFilename() string {
	return path.Join(sf.root, tai64n.Now().Label())
}

func (sf *Spool) pruneOldFiles() {
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

	if count <= sf.MaxFiles {
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

func (sf *Spool) Receive(m *cypress.Message) error {
	cnt, err := sf.enc.Encode(m)
	if err != nil {
		return err
	}

	sf.file.Sync()

	sf.bytes += int64(cnt)

	if sf.bytes >= sf.PerFileSize {
		err := sf.rotate()
		if err != nil {
			return err
		}
	}

	return nil
}

func (sf *Spool) rotate() error {
	sf.file.Close()
	os.Rename(sf.current, sf.newFilename())

	sf.pruneOldFiles()

	err := sf.openCurrent()
	if err != nil {
		return err
	}

	return nil
}

func (s *Spool) Generator() (*SpoolGenerator, error) {
	ents, err := ioutil.ReadDir(s.root)
	if err != nil {
		return nil, err
	}

	var names []string

	for _, e := range ents {
		if e.Name() == "current" {
			continue
		}

		names = append(names, e.Name())
	}

	sort.Strings(names)

	names = append(names, "current")

	// we open all the files up front because something might rotate them
	// out and delete them, so we want to be sure we've still got access
	// to them.
	var files []*os.File

	for _, name := range names {
		f, err := os.Open(filepath.Join(s.root, name))
		if err == nil {
			files = append(files, f)
		}
	}

	dec := cypress.NewStreamDecoder(files[0])
	err = dec.Init()
	if err != nil {
		return nil, err
	}

	return &SpoolGenerator{files: files, dec: dec}, nil
}

type SpoolGenerator struct {
	closed  bool
	files   []*os.File
	current int

	dec *cypress.StreamDecoder
}

var _ = cypress.Generator(&SpoolGenerator{})

func (sg *SpoolGenerator) Generate() (*cypress.Message, error) {
	if sg.closed {
		return nil, io.EOF
	}

	for {
		m, err := sg.dec.Generate()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}

			sg.current++

			if sg.current == len(sg.files) {
				sg.closed = true
				return nil, io.EOF
			}

			dec := cypress.NewStreamDecoder(sg.files[sg.current])
			err = dec.Init()
			if err != nil {
				return nil, err
			}

			sg.dec = dec

			continue
		}

		return m, nil
	}
}

func (sg *SpoolGenerator) Close() error {
	for _, file := range sg.files {
		file.Close()
	}

	return nil
}
