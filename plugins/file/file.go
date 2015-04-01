package file

import (
	"io"
	"os"
	"path/filepath"

	"github.com/vektra/cypress"
	"github.com/vektra/tai64n"
	"github.com/vektra/tail"
)

type File struct {
	path   string
	name   string
	config tail.Config
	tail   *tail.Tail

	lastOffset int64
}

func NewFile(path string, offset int64) (*File, error) {
	cfg := tail.Config{
		Logger: tail.DiscardingLogger,
	}

	if offset > 0 {
		cfg.Location = &tail.SeekInfo{Offset: offset, Whence: os.SEEK_SET}
	}

	t, err := tail.TailFile(path, cfg)
	if err != nil {
		return nil, err
	}

	return &File{path: path, name: filepath.Base(path), config: cfg, tail: t}, nil
}

func NewFollowFile(path string, offset int64) (*File, error) {
	cfg := tail.Config{
		ReOpen: true,
		Follow: true,
		Logger: tail.DiscardingLogger,
	}

	if offset > 0 {
		cfg.Location = &tail.SeekInfo{Offset: offset, Whence: os.SEEK_SET}
	}

	t, err := tail.TailFile(path, cfg)
	if err != nil {
		return nil, err
	}

	return &File{path: path, name: filepath.Base(path), config: cfg, tail: t}, nil
}

func (f *File) Tell() (int64, error) {
	return f.lastOffset, nil
}

func (f *File) Close() error {
	return f.tail.Stop()
}

func (f *File) Generate() (*cypress.Message, error) {
	line := <-f.tail.Lines

	if line == nil {
		return nil, io.EOF
	}

	f.lastOffset = line.Offset + line.Size

	m := cypress.Log()
	m.Timestamp = tai64n.FromTime(line.Time)
	m.AddTag("source", f.name)
	m.Add("message", line.Text)

	return m, nil
}

func (f *File) GenerateLine() (*tail.Line, error) {
	line := <-f.tail.Lines

	if line == nil {
		return nil, io.EOF
	}

	return line, nil
}
