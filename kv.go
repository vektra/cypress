package cypress

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"text/scanner"

	"github.com/gogo/protobuf/proto"
	"github.com/vektra/tai64n"
)

var EParseError = errors.New("Unable to parse line")

const whitespace = 1<<'\t' | 1<<' '
const tokens = scanner.ScanIdents | scanner.ScanInts | scanner.ScanStrings

type KVStream struct {
	Src    io.Reader
	Out    Reciever
	Bare   bool
	Source string
}

func (kvs *KVStream) skipToStart(s *scanner.Scanner) rune {
	if kvs.Bare {
		var buf bytes.Buffer

		for {
			switch tok := s.Next(); tok {
			case '\n':
				if buf.Len() > 0 {
					m := Log()
					if kvs.Source != "" {
						m.AddString("service", kvs.Source)
					}

					m.AddString("message", buf.String())
					kvs.Out.Read(m)

					buf.Reset()
				}

				tok = s.Peek()

				if tok == '>' || tok == scanner.EOF {
					s.Next() // consume the >
					return tok
				}

			case scanner.EOF:
				if buf.Len() > 0 {
					m := Log()

					if kvs.Source != "" {
						m.AddString("source", kvs.Source)
					}

					m.AddString("message", buf.String())
					kvs.Out.Read(m)

				}
				return scanner.EOF

			default:
				buf.WriteString(string(tok))
			}
		}
	} else {
		for {
			switch tok := s.Next(); tok {
			case '\n':
				tok = s.Peek()

				if tok == '>' || tok == scanner.EOF {
					s.Next() // consume the >
					return tok
				}

			case scanner.EOF:
				return scanner.EOF
			}
		}
	}
}

func (kvs *KVStream) skipToNewline(s *scanner.Scanner) {
	for {
		tok := s.Scan()

		if tok == '\n' || tok == scanner.EOF {
			return
		}
	}
}

var cToNano = []int64{100000000, 10000000, 1000000, 100000, 10000, 1000,
	100, 10, 1}

func (s *KVStream) Parse() error {
	var scan scanner.Scanner

	scan.Init(s.Src)
	scan.Whitespace = whitespace
	scan.Mode = tokens

	// Scan the input looking for lines that start with >
	for {
		tok := scan.Peek()

		if tok != '>' {
			tok = s.skipToStart(&scan)
		} else {
			scan.Next() // consume the >
		}

		if tok == scanner.EOF {
			return io.EOF
		}

		// We're at the start of a message now

		m := Log()
		if s.Source != "" {
			m.AddString("source", s.Source)
		}

		// Detect a type flag
		switch scan.Peek() {
		case '!':
			scan.Next()
			m.Type = proto.Uint32(tMetric)
		case '$':
			scan.Next()
			m.Type = proto.Uint32(tTrace)
		case '+':
			scan.Next()
			m.Type = proto.Uint32(tAudit)
		}

		if scan.Next() != ' ' {
			s.skipToNewline(&scan)
			continue
		}

		// Detect a predeclared timestamp
		if scan.Peek() == '@' {
			scan.Next() // consume the @

			var buf bytes.Buffer

			buf.WriteString("@")

			for {
				tok := scan.Next()

				if tok == ' ' {
					ts := tai64n.ParseTAI64NLabel(buf.String())

					if ts != nil {
						m.Timestamp = ts
					}

					break
				} else {
					buf.WriteString(string(tok))
				}
			}
		}

		if scan.Peek() == '\\' {
			scan.Next() // consume the \

			var buf bytes.Buffer

			for {
				tok := scan.Next()

				if tok == ' ' {
					m.SessionId = proto.String(buf.String())
					break
				} else {
					buf.WriteString(string(tok))
				}
			}
		}

		// Pull out a key=val sequence
		for {
			tok = scan.Peek()

			var key string

			if tok == '\n' || tok == scanner.EOF {
				if m != nil {
					s.Out.Read(m)
				}

				break
			}

			tok = scan.Scan()

			if tok != scanner.Ident {
				goto bad
			}

			key = scan.TokenText()

			if scan.Scan() != '=' {
				goto bad
			}

			switch scan.Scan() {
			case ':':
				st := scan.Scan()

				if st != scanner.Int {
					goto bad
				}

				tsec := scan.TokenText()

				st = scan.Scan()

				if st != '.' {
					goto bad
				}

				st = scan.Scan()

				if st != scanner.Int {
					goto bad
				}

				tssec := scan.TokenText()

				sec, _ := strconv.ParseInt(tsec, 10, 64)
				subsec, _ := strconv.ParseInt(tssec, 10, 32)

				if len(tssec) <= 9 {
					subsec *= cToNano[len(tssec)-1]
				}

				m.AddInterval(key, uint64(sec), uint32(subsec))
			case scanner.Int:
				i, err := strconv.ParseInt(scan.TokenText(), 0, 64)
				if err != nil {
					goto bad
				}

				m.AddInt(key, i)
			case scanner.String, scanner.RawString:
				s := scan.TokenText()

				m.AddString(key, s[1:len(s)-1])
			case scanner.Ident, scanner.Float, scanner.Char:
				m.AddString(key, scan.TokenText())

			default:
				goto bad
			}

			continue

		bad:
			s.skipToNewline(&scan)

			break
		}
	} // for each line

	return nil
}

func ParseKVStream(in io.Reader, r Reciever) {
	s := KVStream{in, r, false, ""}
	s.Parse()
}

type MessageBuffer struct {
	Messages []*Message
}

func (b *MessageBuffer) Read(m *Message) (err error) {
	b.Messages = append(b.Messages, m)
	return nil
}

func ParseKV(line string) (*Message, error) {
	buf := bytes.NewReader([]byte(line))

	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) == 0 {
		return nil, EParseError
	}

	return mbuf.Messages[0], nil
}
