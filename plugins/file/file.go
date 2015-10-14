package file

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vektra/cypress"
	"github.com/vektra/tai64n"

	"gopkg.in/fsnotify.v1"
	"gopkg.in/tomb.v2"
)

type Line struct {
	Line   string
	Offset int64
	Time   time.Time
}

func (l *Line) Next() int64 {
	return l.Offset + int64(len(l.Line))
}

type File struct {
	path string
	name string

	lastOffset int64

	t tomb.Tomb

	offset int64
	lines  chan Line
}

func NewFile(path string, offset int64) (*File, error) {
	f := &File{
		path:   path,
		name:   filepath.Base(path),
		lines:  make(chan Line),
		offset: offset,
	}

	f.t.Go(f.readOnce)

	return f, nil
}

func (f *File) readOnce() error {
	defer close(f.lines)

	r, err := os.Open(f.path)

	if err != nil {
		return err
	}

	defer r.Close()

	_, err = r.Seek(f.offset, os.SEEK_SET)
	if err != nil {
		return err
	}

	buf := bufio.NewReader(r)

	offset := f.offset

	for {
		str, err := buf.ReadString('\n')
		if err != nil {
			return err
		}

		select {
		case f.lines <- Line{str, offset, time.Now()}:
		case <-f.t.Dying():
			return nil
		}

		offset += int64(len(str))
	}

	return nil
}

func NewFollowFile(path string, offset int64) (*File, error) {
	f := &File{
		path:   path,
		name:   filepath.Base(path),
		lines:  make(chan Line),
		offset: offset,
	}

	f.t.Go(f.readFollow)

	return f, nil
}

func (f *File) readFollow() error {
	defer close(f.lines)

	r, err := os.Open(f.path)

	if err != nil {
		return err
	}

	defer r.Close()

	_, err = r.Seek(f.offset, os.SEEK_SET)
	if err != nil {
		return err
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	defer w.Close()

	w.Add(f.path)
	w.Add(filepath.Dir(f.path))

	buf := bufio.NewReader(r)

	offset := f.offset

top:
	for {
		str, err := buf.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return err
			}

			for {
				select {
				case evt := <-w.Events:
					if evt.Op&fsnotify.Write == fsnotify.Write {
						continue top
					}

					if evt.Op&fsnotify.Create == fsnotify.Create {
						if evt.Name == f.path {
							r.Close()

							r, err = os.Open(f.path)
							if err != nil {
								return err
							}

							buf = bufio.NewReader(r)

							offset = 0

							continue top
						}
					}
				case err := <-w.Errors:
					return err
				case <-f.t.Dying():
					return nil
				}
			}
		}

		select {
		case f.lines <- Line{str, offset, time.Now()}:
			// nothing

		case <-f.t.Dying():
			return nil
		}

		offset += int64(len(str))
	}

	return nil
}

func (f *File) Tell() (int64, error) {
	return f.lastOffset, nil
}

func (f *File) Close() error {
	f.t.Kill(nil)
	f.t.Wait()

	return nil
}

func (f *File) Generate() (*cypress.Message, error) {
	line, err := f.GenerateLine()
	if err != nil {
		return nil, err
	}

	f.lastOffset = line.Offset + int64(len(line.Line))

	m := cypress.Log()
	m.Timestamp = tai64n.FromTime(line.Time)
	m.AddTag("source", f.name)
	m.Add("message", strings.TrimSpace(line.Line))

	return m, nil
}

func (f *File) Lines() chan Line {
	return f.lines
}

func (f *File) GenerateLine() (*Line, error) {
	line, ok := <-f.lines
	if !ok {
		return nil, io.EOF
	}

	return &line, nil
}
