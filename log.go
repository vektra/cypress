package cypress

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/gogo/protobuf/proto"
	"github.com/mgutz/ansi"
	"github.com/vektra/errors"
	"github.com/vektra/tai64n"
)

var PresetKeys = map[string]uint32{
	"message": 1,
	"value":   2,
	"source":  3,
}

var PresetKeysFromIndex = map[uint32]string{
	1: "message",
	2: "value",
	3: "source",
}

const LOG = 0
const METRIC = 1
const TRACE = 2
const AUDIT = 3

func Log() *Message {
	return &Message{Timestamp: tai64n.Now(), Type: proto.Uint32(LOG)}
}

func Metric() *Message {
	return &Message{Timestamp: tai64n.Now(), Type: proto.Uint32(METRIC)}
}

func Trace() *Message {
	return &Message{Timestamp: tai64n.Now(), Type: proto.Uint32(TRACE)}
}

func Audit() *Message {
	return &Message{Timestamp: tai64n.Now(), Type: proto.Uint32(AUDIT)}
}

func (m *Message) StringType() string {
	switch m.GetType() {
	case LOG:
		return "log"
	case METRIC:
		return "metric"
	case TRACE:
		return "trace"
	case AUDIT:
		return "audit"
	default:
		return "unknown"
	}
}

func (m *Message) ProtoShow() {
	proto.MarshalText(os.Stdout, m)
	os.Stdout.Write([]byte("\n"))
}

func subsecond(s uint32) string {
	str := strconv.FormatInt(int64(s), 10)

	for len(str) < 9 {
		str = "0" + str
	}

	for {
		if str[len(str)-1] == '0' {
			str = str[:len(str)-1]
		} else {
			break
		}
	}

	if str == "" {
		return "0"
	}

	return str
}

func strquote(in string) string {
	return strings.Replace(in, `"`, `\"`, -1)
}

func (m *Message) KVPairs() string {
	var buf bytes.Buffer

	m.KVPairsInto(&buf)

	return buf.String()
}

func (m *Message) KVPairsInto(buf *bytes.Buffer) {
	for _, attr := range m.Attributes {
		buf.WriteString(" ")
		buf.WriteString(attr.StringKey())
		buf.WriteString("=")

		switch {
		case attr.Ival != nil:
			buf.WriteString(strconv.FormatInt(*attr.Ival, 10))
		case attr.Fval != nil:
			buf.WriteString(strconv.FormatFloat(*attr.Fval, 'g', -1, 64))
		case attr.Sval != nil:
			buf.WriteString("\"")
			buf.WriteString(strquote(*attr.Sval))
			buf.WriteString("\"")
		case attr.Bval != nil:
			buf.WriteString("\"")
			buf.WriteString(strquote(string(attr.Bval)))
			buf.WriteString("\"")
		case attr.Tval != nil:
			buf.WriteString(":")
			buf.WriteString(strconv.FormatInt(int64(attr.Tval.GetSeconds()), 10))
			buf.WriteString(".")
			buf.WriteString(subsecond(attr.Tval.GetNanoseconds()))
		}
	}
}

func (m *Message) KVString() string {
	var buf bytes.Buffer

	m.KVStringInto(&buf)

	return buf.String()
}

func (m *Message) KVStringInto(buf *bytes.Buffer) {
	buf.WriteString(">")
	switch {
	case m.GetType() == METRIC:
		buf.WriteString("! ")
	case m.GetType() == TRACE:
		buf.WriteString("$ ")
	case m.GetType() == AUDIT:
		buf.WriteString("* ")
	default:
		buf.WriteString(" ")
	}

	buf.WriteString(m.GetTimestamp().Label())

	if s := m.GetSessionId(); len(s) > 0 {
		buf.WriteString(" \\")
		buf.WriteString(s)
	}

	m.KVPairsInto(buf)
}

var voltColor = ansi.ColorCode("blue")
var systemColor = ansi.ColorCode("yellow")
var resetColor = ansi.ColorCode("reset")

func (m *Message) SyslogString(colorize bool, align bool) string {
	var buf bytes.Buffer

	// Special case the logs that come out of the volts to make them
	// easier to read
	if m.GetType() == 0 {
		if volt, ok := m.GetString("volt"); ok {
			if log, ok := m.GetString("log"); ok {
				if colorize {
					buf.WriteString(voltColor)
				}

				time := m.GetTimestamp().Time().Format(time.RFC3339Nano)

				buf.WriteString(time)

				if align {
					for i := len(time); i < 35; i++ {
						buf.WriteString(" ")
					}
				}

				buf.WriteString(" ")

				if s := m.GetSessionId(); len(s) > 0 {
					buf.WriteString(uuid.UUID(s).String()[0:7])
				} else {
					buf.WriteString("0000000")
				}

				buf.WriteString(" ")
				buf.WriteString(volt)
				buf.WriteString(" ")
				if colorize {
					buf.WriteString(resetColor)
				}
				buf.WriteString(log)
				return buf.String()
			}
		}
	}

	if colorize {
		buf.WriteString(systemColor)
	}

	time := m.GetTimestamp().Time().Format(time.RFC3339Nano)

	buf.WriteString(time)

	if align {
		for i := len(time); i < 35; i++ {
			buf.WriteString(" ")
		}
	}

	buf.WriteString(" ")

	if s := m.GetSessionId(); len(s) > 0 {
		buf.WriteString(uuid.UUID(s).String()[0:7])
	} else {
		buf.WriteString("0000000")
	}

	buf.WriteString(" system ")

	if colorize {
		buf.WriteString(resetColor)
	}

	if m.GetType() == METRIC {
		buf.WriteString("!")
	} else if m.GetType() == TRACE {
		buf.WriteString("$")
	} else {
		buf.WriteString("*")
	}

	buf.WriteString(m.KVPairs())

	return buf.String()
}

func (m *Message) HumanString() string {
	var buf bytes.Buffer

	if m.GetType() == METRIC {
		buf.WriteString("! ")
	} else if m.GetType() == TRACE {
		buf.WriteString("$ ")
	} else if m.GetType() == AUDIT {
		buf.WriteString("* ")
	} else {
		buf.WriteString(" ")
	}

	buf.WriteString(" ")
	buf.WriteString(m.GetTimestamp().String())
	buf.WriteString(" ")

	if s := m.GetSessionId(); len(s) > 0 {
		buf.WriteString(s[:7])
	} else {
		buf.WriteString("0000000")
	}

	buf.WriteString(m.KVPairs())

	return buf.String()
}

func (a *Attribute) StringKey() string {
	if a.Key != 0 {
		return PresetKeysFromIndex[a.Key]
	}

	if a.Skey == nil {
		return "<nil>"
	}

	return *a.Skey
}

func (m *Message) Get(key string) (interface{}, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey() == key {
			if attr.Ival != nil {
				return *attr.Ival, true
			}

			if attr.Fval != nil {
				return *attr.Fval, true
			}

			if attr.Sval != nil {
				return *attr.Sval, true
			}

			if attr.Bval != nil {
				return attr.Bval, true
			}

			if attr.Tval != nil {
				return attr.Tval, true
			}

			if attr.Boolval != nil {
				return *attr.Boolval, true
			}
		}
	}

	return nil, false
}

func (m *Message) GetInt(key string) (int64, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey() == key {
			if attr.Ival == nil {
				return 0, false
			}

			return *attr.Ival, true
		}
	}

	return 0, false
}

func (m *Message) GetFloat(key string) (float64, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey() == key {
			if attr.Fval == nil {
				return 0, false
			}

			return *attr.Fval, true
		}
	}

	return 0, false
}

func (m *Message) GetString(key string) (string, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey() == key {
			if attr.Sval == nil {
				return "", false
			}

			return *attr.Sval, true
		}
	}

	return "", false
}

func (m *Message) GetBytes(key string) ([]byte, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey() == key {
			if attr.Bval == nil {
				return nil, false
			}

			return attr.Bval, true
		}
	}

	return nil, false
}

func (m *Message) GetInterval(key string) (*Interval, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey() == key {
			if attr.Tval == nil {
				return nil, false
			}

			return attr.Tval, true
		}
	}

	return nil, false
}

func (m *Message) GetBool(key string) (bool, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey() == key {
			if attr.Boolval == nil {
				return false, false
			}

			return *attr.Boolval, true
		}
	}

	return false, false
}

func (m *Message) For(id string) {
	m.SessionId = &id
}

var ErrBadValue = errors.New("Invalid type for attribute value")

type Inter interface {
	Int() int64
}

type Stringer interface {
	String() string
}

func (m *Message) Add(key string, val interface{}) error {
	attr := &Attribute{}

	if val, ok := PresetKeys[key]; ok {
		attr.Key = val
	} else {
		attr.Skey = &key
	}

	switch x := val.(type) {
	case bool:
		attr.Boolval = &x
	case int:
		bi := int64(x)
		attr.Ival = &bi
	case int32:
		bi := int64(x)
		attr.Ival = &bi
	case uint32:
		bi := int64(x)
		attr.Ival = &bi
	case uint64:
		bi := int64(x)
		attr.Ival = &bi
	case int64:
		bi := x
		attr.Ival = &bi
	case float32:
		bi := float64(x)
		attr.Fval = &bi
	case float64:
		bi := x
		attr.Fval = &bi
	case Inter:
		t := x.Int()
		attr.Ival = &t
	case string:
		attr.Sval = &x
	case time.Duration:
		attr.Tval = durationToInterval(x)
	case Stringer:
		t := x.String()
		attr.Sval = &t
	case []byte:
		attr.Bval = x
	case error:
		t := x.Error()
		attr.Sval = &t
	default:
		return m.tryContainers(key, x)
	}

	m.Attributes = append(m.Attributes, attr)
	return nil
}

func (m *Message) tryContainers(prefix string, x interface{}) error {
	v := reflect.ValueOf(x)

	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			key := fmt.Sprintf("%s.%d", prefix, i)
			m.Add(key, v.Index(i).Interface())
		}

		return nil
	case reflect.Map:
		keys := v.MapKeys()

		for _, key := range keys {
			m.Add(fmt.Sprintf("%s.%s", prefix, key), v.MapIndex(key).Interface())
		}

		return nil
	case reflect.Struct:
		for idx, key := range fieldNames(v) {
			m.Add(fmt.Sprintf("%s.%s", prefix, key), v.Field(idx).Interface())
		}

		return nil
	}

	return errors.Subject(ErrBadValue, v.Type())
}

// Lovingly lifted from https://github.com/codahale/lunk/blob/master/reflect.go#L60

func fieldNames(v reflect.Value) map[int]string {
	t := v.Type()

	// check to see if a cached set exists
	cachedFieldNamesRW.RLock()
	m, ok := cachedFieldNames[t]
	cachedFieldNamesRW.RUnlock()

	if ok {
		return m
	}

	// otherwise, create it and return it
	cachedFieldNamesRW.Lock()
	m = make(map[int]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		fld := t.Field(i)
		if fld.PkgPath != "" {
			continue // ignore all unexpected fields
		}

		name := fld.Tag.Get("log")
		if name == "" {
			name = strings.ToLower(fld.Name)
		}
		m[i] = name
	}
	cachedFieldNames[t] = m
	cachedFieldNamesRW.Unlock()
	return m
}

var (
	cachedFieldNames   = make(map[reflect.Type]map[int]string, 20)
	cachedFieldNamesRW = new(sync.RWMutex)
)

func (m *Message) AddMany(vals ...interface{}) error {
	if len(vals)%2 != 0 {
		panic("Passed an uneven number of values to Send")
	}

	for i := 0; i < len(vals); i += 2 {
		var key string

		if k, ok := vals[i].(string); ok {
			key = k
		} else {
			key = fmt.Sprintf("%s", vals[i])
		}

		m.Add(key, vals[i+1])
	}

	return nil
}

func (m *Message) AddInt(key string, val int64) error {
	attr := &Attribute{}

	if val, ok := PresetKeys[key]; ok {
		attr.Key = val
	} else {
		attr.Skey = &key
	}

	attr.Ival = &val

	m.Attributes = append(m.Attributes, attr)
	return nil
}

func (m *Message) AddFloat(key string, val float64) error {
	attr := &Attribute{}

	if val, ok := PresetKeys[key]; ok {
		attr.Key = val
	} else {
		attr.Skey = &key
	}

	attr.Fval = &val

	m.Attributes = append(m.Attributes, attr)
	return nil
}

func (m *Message) AddString(key string, val string) error {
	attr := &Attribute{}

	if val, ok := PresetKeys[key]; ok {
		attr.Key = val
	} else {
		attr.Skey = &key
	}

	attr.Sval = &val

	m.Attributes = append(m.Attributes, attr)
	return nil
}

func (m *Message) AddBytes(key string, val []byte) error {
	attr := &Attribute{}

	if val, ok := PresetKeys[key]; ok {
		attr.Key = val
	} else {
		attr.Skey = &key
	}

	attr.Bval = val

	m.Attributes = append(m.Attributes, attr)
	return nil
}

func (m *Message) AddInterval(key string, sec uint64, nsec uint32) error {
	attr := &Attribute{}

	if val, ok := PresetKeys[key]; ok {
		attr.Key = val
	} else {
		attr.Skey = &key
	}

	iv := &Interval{
		Seconds:     sec,
		Nanoseconds: nsec,
	}

	attr.Tval = iv

	m.Attributes = append(m.Attributes, attr)
	return nil
}

func durationToInterval(dur time.Duration) *Interval {
	total_nsec := uint64(dur.Nanoseconds())

	sec := total_nsec / uint64(time.Second)
	nsec := total_nsec % uint64(time.Second)

	return &Interval{
		Seconds:     sec,
		Nanoseconds: uint32(nsec),
	}
}

func (i *Interval) Duration() time.Duration {
	return (time.Duration(i.Seconds) * time.Second) +
		(time.Duration(i.Nanoseconds) * time.Nanosecond)
}

func (m *Message) AddDuration(key string, dur time.Duration) error {
	attr := &Attribute{}

	if val, ok := PresetKeys[key]; ok {
		attr.Key = val
	} else {
		attr.Skey = &key
	}

	attr.Tval = durationToInterval(dur)

	m.Attributes = append(m.Attributes, attr)
	return nil
}
