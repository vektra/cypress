package cypress

import (
	"bytes"
	"io"
	"strconv"
	"strings"

	"github.com/vektra/cypress/scanner"

	"github.com/gogo/protobuf/proto"
	"github.com/vektra/tai64n"
)

const whitespace = 1<<'\t' | 1<<' '
const tokens = scanner.ScanIdents | scanner.ScanFloats | scanner.ScanStrings

var cToNano = []int64{100000000, 10000000, 1000000, 100000, 10000, 1000,
	100, 10, 1}

// Given a reader, parse it as KV lines and send the message to r
func ParseKVStream(in io.Reader, r Receiver) error {
	parser := NewKVParser(in)

	return Glue(parser, r)
}

// Parse the given line in KV format into a Message
func ParseKV(line string) (*Message, error) {
	buf := bytes.NewReader([]byte(line))

	parser := NewKVParser(buf)

	return parser.Generate()
}

// A type which can transform a io.Reader into a set of Messages
type KVParser struct {
	Bare bool

	r    io.Reader
	scan scanner.Scanner
}

func (kv *KVParser) readBare() (*Message, error) {
	var buf bytes.Buffer

	for {
		switch tok := kv.scan.Next(); tok {
		case '\n':
			if buf.Len() > 0 {
				m := Log()
				m.AddString("message", buf.String())

				return m, nil
			}
		case scanner.EOF:
			if buf.Len() > 0 {
				m := Log()

				m.AddString("message", buf.String())

				return m, nil
			}

			return nil, io.EOF

		default:
			buf.WriteString(string(tok))
		}
	}

	return nil, io.EOF
}

func (kv *KVParser) skipToStart() rune {
	for {
		switch tok := kv.scan.Next(); tok {
		case '\n':
			tok = kv.scan.Peek()

			switch tok {
			case '>':
				return kv.scan.Next()
			case scanner.EOF:
				return tok
			}

		case scanner.EOF:
			return scanner.EOF
		}
	}
}

func (kv *KVParser) skipToNewline() {
	for {
		tok := kv.scan.Scan()

		if tok == '\n' || tok == scanner.EOF {
			return
		}
	}
}

// Create a new KVParser from the data in r
func NewKVParser(r io.Reader) *KVParser {
	kv := &KVParser{r: r}

	kv.scan.Init(r)
	kv.scan.Whitespace = whitespace
	kv.scan.Mode = tokens

	return kv
}

// Read and decode and return another Message
func (s *KVParser) Generate() (*Message, error) {
	scan := &s.scan

restart:
	tok := scan.Peek()

	if tok == scanner.EOF {
		return nil, io.EOF
	}

	if tok != '>' {
		if s.Bare {
			return s.readBare()
		}

		tok = s.skipToStart()

		if tok == scanner.EOF {
			return nil, io.EOF
		}
	} else {
		scan.Next() // consume the >
	}

	// We're at the start of a message now

	m := Log()

	tok = scan.Peek()

	// Detect a type flag
	switch tok {
	case '!':
		scan.Next()
		m.Type = proto.Uint32(METRIC)
	case '$':
		scan.Next()
		m.Type = proto.Uint32(TRACE)
	case '*':
		scan.Next()
		m.Type = proto.Uint32(AUDIT)
	case '?':
		scan.Next()
		m.Type = proto.Uint32(HEARTBEAT)
	case scanner.EOF:
		return nil, io.EOF
	}

	if scan.Next() != ' ' {
		goto restart
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

	if scan.Peek() == '[' {
		scan.Next()

		for {
			var name, value string

			tok := scan.Scan()

			if tok == ']' {
				break
			}

			if tok == '!' {
				tok = scan.Scan()

				if tok != scanner.Ident {
					goto badtag
				}

				name = scan.TokenText()

				m.Tags = append(m.Tags, &Tag{Name: name})

				continue
			}

			switch tok {
			case scanner.Ident:
				name = scan.TokenText()

				for scan.Peek() == '.' {
					scan.Scan()

					tok = scan.Scan()

					if tok != scanner.Ident {
						goto badtag
					}

					name = name + "." + scan.TokenText()
				}
			case scanner.String, scanner.RawString:
				name = scan.TokenText()
				name = name[1 : len(name)-1]
			default:
				goto badtag
			}

			tok = scan.Scan()

			if tok != '=' {
				goto badtag
			}

			tok = scan.Scan()

			switch tok {
			case scanner.String, scanner.RawString:
				value = scan.TokenText()

				value = value[1 : len(value)-1]
			case scanner.Ident:
				value = scan.TokenText()
			default:
				goto badtag
			}

			m.Tags = append(m.Tags, &Tag{Name: name, Value: &value})

			continue

		badtag:
			s.skipToNewline()

			goto restart
		}
	}

	// Pull out a key=val sequence
	for {
		tok = scan.Peek()

		var key string

		if tok == '\n' || tok == scanner.EOF {
			return m, nil
		}

		tok = scan.Scan()

		switch tok {
		case scanner.Ident:
			key = scan.TokenText()

			for scan.Peek() == '.' {
				scan.Scan()

				tok = scan.Scan()

				if tok != scanner.Ident {
					goto bad
				}

				key = key + "." + scan.TokenText()
			}
		case scanner.String, scanner.RawString:
			key = scan.TokenText()
			key = key[1 : len(key)-1]
		default:
			goto bad
		}

		if scan.Scan() != '=' {
			goto bad
		}

		switch scan.Scan() {
		case ':':
			st := scan.Scan()

			if st != scanner.Float {
				goto bad
			}

			dec := scan.TokenText()

			dot := strings.IndexByte(dec, '.')

			if dot == -1 {
				goto bad
			}

			tsec := dec[:dot]
			tssec := dec[dot+1:]

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
		case scanner.Float:
			i, err := strconv.ParseFloat(scan.TokenText(), 64)
			if err != nil {
				i, err := strconv.ParseInt(scan.TokenText(), 0, 64)
				if err != nil {
					goto bad
				}

				m.AddInt(key, i)
			} else {
				m.AddFloat(key, i)
			}
		case scanner.String, scanner.RawString:
			s := scan.TokenText()

			m.AddString(key, s[1:len(s)-1])
		case scanner.Ident, scanner.Char:
			m.AddString(key, scan.TokenText())

		default:
			goto bad
		}

		continue

	bad:
		s.skipToNewline()

		goto restart
	}

	return nil, io.EOF
}

// To fit the Generator interface
func (p *KVParser) Close() error {
	return nil
}
