package file

import (
	"io"
	"strings"
	"sync"

	"github.com/vektra/cypress"
	"github.com/vektra/tai64n"
)

type inputLine struct {
	f    *File
	line *Line
	path string
}

type Monitor struct {
	db      *OffsetDB
	files   []*File
	lines   chan inputLine
	filewg  sync.WaitGroup
	offsets map[string]int64

	shutdown chan bool
	done     chan bool

	Debug bool
}

func NewMonitor() *Monitor {
	return &Monitor{
		shutdown: make(chan bool, 1),
		done:     make(chan bool),
	}
}

func (m *Monitor) OpenOffsetDB(path string) error {
	db, err := NewOffsetDB(path)
	if err != nil {
		return err
	}

	m.db = db

	return nil
}

func (m *Monitor) OpenFiles(once bool, args []string) error {
	m.lines = make(chan inputLine, len(args))
	m.offsets = map[string]int64{}

	var err error

	for _, path := range args {
		var offset int64
		var f *File

		if m.db != nil {
			entry, err := m.db.Get(path)
			if err != nil {
				return err
			}

			if entry != nil && entry.Valid() {
				offset = entry.Offset
			}
		}

		if once {
			f, err = NewFile(path, offset)
		} else {
			f, err = NewFollowFile(path, offset)
		}

		if err != nil {
			return err
		}

		if m.Debug {
			dbgLog.Printf("Watching '%s' from offset '%d'", path, offset)
		}

		m.offsets[path] = offset

		m.files = append(m.files, f)

		m.filewg.Add(1)
		go func(path string) {
			defer m.filewg.Done()

			for {
				line, err := f.GenerateLine()
				if err != nil {
					if m.Debug && err != io.EOF {
						dbgLog.Printf("Error reading files from '%s': %s", path, err)
					}

					return
				}

				m.lines <- inputLine{f, line, path}
			}
		}(path)
	}

	return nil
}

func (m *Monitor) WatchFiles() {
	m.filewg.Wait()
	close(m.lines)
}

func (m *Monitor) SignalShutdown() {
	// Don't block if we can't add anything to shutdown, meaning
	// there is a shutdown pending anyway.

	select {
	case m.shutdown <- true:
	default:
	}
}

func (m *Monitor) WaitShutdown() {
	m.SignalShutdown()
	<-m.done
}

func (m *Monitor) CloseFiles() error {
	for _, f := range m.files {
		f.Close()
	}

	m.filewg.Wait()
	return nil
}

func (m *Monitor) FlushOffsets() {
	if m.db == nil {
		return
	}

	for path, offset := range m.offsets {
		if m.Debug {
			dbgLog.Printf("Remembering offset of '%s' as '%d'", path, offset)
		}

		m.db.Set(path, offset)
	}
}

func (m *Monitor) finished() {
	close(m.done)
	m.FlushOffsets()
}

func (m *Monitor) Run(enc cypress.Receiver) error {
	go m.WatchFiles()

	defer m.finished()

	for {
		select {
		case <-m.shutdown:
			if m.Debug {
				dbgLog.Printf("Shutting down")
			}

			return m.CloseFiles()
		case il := <-m.lines:
			if il.line == nil {
				m.SignalShutdown()
				continue
			}

			msg := cypress.Log()
			msg.Timestamp = tai64n.FromTime(il.line.Time)
			msg.AddTag("source", il.f.name)
			msg.Add("message", strings.TrimSpace(il.line.Line))

			err := enc.Receive(msg)
			if err != nil {
				if m.Debug {
					dbgLog.Printf("Error sending message: %s", err)
				}

				m.CloseFiles()
				return err
			}

			m.offsets[il.path] = il.line.Next()
		}
	}

	return nil
}
