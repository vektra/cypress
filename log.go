package cypress

import (
	"bytes"
	"fmt"
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

// Type code representing a generic log message
const LOG = 0

// Type code for a metric
const METRIC = 1

// Type code for an application trace
const TRACE = 2

// Type code for an audit (ie, high value log) message
const AUDIT = 3

// Type code for a heartbeat, representing liveness.
const HEARTBEAT = 4

// The default version of messages generated
const DEFAULT_VERSION = 1

// Create a new Message
func NewMessage() *Message {
	return &Message{Version: DEFAULT_VERSION, Timestamp: tai64n.Now()}
}

// Create a new Message of type LOG
func Log() *Message {
	return &Message{Version: DEFAULT_VERSION, Timestamp: tai64n.Now(), Type: proto.Uint32(LOG)}
}

// Create a new Message of type METRIC
func Metric() *Message {
	return &Message{Version: DEFAULT_VERSION, Timestamp: tai64n.Now(), Type: proto.Uint32(METRIC)}
}

// Create a new Message of type TRACE
func Trace() *Message {
	return &Message{Version: DEFAULT_VERSION, Timestamp: tai64n.Now(), Type: proto.Uint32(TRACE)}
}

// Create a new Message of type AUDIT
func Audit() *Message {
	return &Message{Version: DEFAULT_VERSION, Timestamp: tai64n.Now(), Type: proto.Uint32(AUDIT)}
}

// Create a new Message of type HEARTBEAT
func Heartbeat() *Message {
	return &Message{Version: DEFAULT_VERSION, Timestamp: tai64n.Now(), Type: proto.Uint32(HEARTBEAT)}
}

// Return the Message type as a string
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
	case HEARTBEAT:
		return "heartbeat"
	default:
		return "unknown"
	}
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

// Return the Message's tags as a string formatted for use in Postgresql HSTORE
func (m *Message) HstoreTags() string {
	var buf bytes.Buffer

	m.HstoreTagsInto(&buf)

	return buf.String()
}

// Write a Message's tags formatted for Postgresql HSTORE into a buffer
func (m *Message) HstoreTagsInto(buf *bytes.Buffer) {
	for i, tag := range m.Tags {
		buf.WriteString("\"")
		buf.WriteString(tag.Name)
		buf.WriteString("\"")
		buf.WriteString("=>")

		buf.WriteString("\"")
		if tag.Value != nil {
			buf.WriteString(strquote(*tag.Value))
		}
		buf.WriteString("\"")

		if i < len(m.Tags)-1 {
			buf.WriteString(", ")
		}
	}
}

// Return the Message's attributes as a string formatted for use in Postgresql HSTORE
func (m *Message) HstoreAttributes() string {
	var buf bytes.Buffer

	m.HstoreAttributesInto(&buf)

	return buf.String()
}

// Write a Message's attributes formatted for Postgresql HSTORE into a buffer
func (m *Message) HstoreAttributesInto(buf *bytes.Buffer) {
	for i, attr := range m.Attributes {
		buf.WriteString("\"")
		buf.WriteString(attr.StringKey(m))
		buf.WriteString("\"")
		buf.WriteString("=>")

		switch {
		case attr.Ival != nil:
			buf.WriteString(strconv.FormatInt(*attr.Ival, 10))
		case attr.Fval != nil:
			buf.WriteString(strconv.FormatFloat(*attr.Fval, 'g', -1, 64))
		case attr.Boolval != nil:
			if *attr.Boolval {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}
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

		if i < len(m.Attributes)-1 {
			buf.WriteString(", ")
		}
	}
}

// Return the Message's attributes as a KV formatted string
func (m *Message) KVPairs() string {
	var buf bytes.Buffer

	m.KVPairsInto(&buf)

	return buf.String()
}

// Write a Message's attributes in KV format to a buffer
func (m *Message) KVPairsInto(buf *bytes.Buffer) {
	for _, attr := range m.Attributes {
		buf.WriteString(" ")
		buf.WriteString(attr.StringKey(m))
		buf.WriteString("=")

		switch {
		case attr.Ival != nil:
			buf.WriteString(strconv.FormatInt(*attr.Ival, 10))
		case attr.Fval != nil:
			buf.WriteString(strconv.FormatFloat(*attr.Fval, 'g', -1, 64))
		case attr.Boolval != nil:
			if *attr.Boolval {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}
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

// Return the Message as a KV formatted string
func (m *Message) KVString() string {
	var buf bytes.Buffer

	m.KVStringInto(&buf)

	return buf.String()
}

// Write a Message in KV format to a buffer
func (m *Message) KVStringInto(buf *bytes.Buffer) {
	buf.WriteString(">")
	switch {
	case m.GetType() == METRIC:
		buf.WriteString("! ")
	case m.GetType() == TRACE:
		buf.WriteString("$ ")
	case m.GetType() == AUDIT:
		buf.WriteString("* ")
	case m.GetType() == HEARTBEAT:
		buf.WriteString("? ")
	default:
		buf.WriteString(" ")
	}

	buf.WriteString(m.GetTimestamp().Label())

	if s := m.GetSessionId(); len(s) > 0 {
		buf.WriteString(" \\")
		buf.WriteString(s)
	}

	m.KVTagsInto(buf)
	m.KVPairsInto(buf)
}

// Write a Message's tags in KV format to a buffer
func (m *Message) KVTagsInto(buf *bytes.Buffer) {
	if len(m.Tags) > 0 {
		buf.WriteString(" [")

		for i := 0; i < len(m.Tags); i++ {
			if m.Tags[i].Value == nil {
				buf.WriteString("!")
				buf.WriteString(m.Tags[i].Name)
			} else {
				buf.WriteString(m.Tags[i].Name)
				buf.WriteString("=")

				buf.WriteString("\"")
				buf.WriteString(strquote(*m.Tags[i].Value))
				buf.WriteString("\"")

				if i < len(m.Tags)-1 {
					buf.WriteString(" ")
				}
			}
		}

		buf.WriteString("]")
	}
}

var voltColor = ansi.ColorCode("blue")
var systemColor = ansi.ColorCode("yellow")
var resetColor = ansi.ColorCode("reset")

// Return a Message formatted as a syslog string.
// colorize indicates if ANSI color codes should be used to highlight portions.
// align controls if time field is aligned to 35 bytes (useful for when
// a set of messages are displayed on lines next to eachother).
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

	m.KVTagsInto(&buf)
	buf.WriteString(m.KVPairs())

	return buf.String()
}

// Return a Message as a string formatted for easy human reading
func (m *Message) HumanString() string {
	var buf bytes.Buffer

	if m.GetType() == METRIC {
		buf.WriteString("! ")
	} else if m.GetType() == TRACE {
		buf.WriteString("$ ")
	} else if m.GetType() == AUDIT {
		buf.WriteString("* ")
	} else if m.GetType() == HEARTBEAT {
		buf.WriteString("? ")
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

	m.KVTagsInto(&buf)
	buf.WriteString(m.KVPairs())

	return buf.String()
}

// Return the key as a string of this Attribute within Message m
func (a *Attribute) StringKey(m *Message) string {
	if a.Key != 0 {
		return versionSymbols[m.Version].FromIndex(a.Key)
	}

	if a.Skey == nil {
		return "<nil>"
	}

	return *a.Skey
}

// Find an Attribute by name and return it's value.
func (m *Message) Get(key string) (interface{}, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey(m) == key {
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

// Find an Attibute by name that is an int and return it
func (m *Message) GetInt(key string) (int64, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey(m) == key {
			if attr.Ival == nil {
				return 0, false
			}

			return *attr.Ival, true
		}
	}

	return 0, false
}

// Find an Attibute by name that is a float and return it
func (m *Message) GetFloat(key string) (float64, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey(m) == key {
			if attr.Fval == nil {
				return 0, false
			}

			return *attr.Fval, true
		}
	}

	return 0, false
}

// Find an Attibute by name that is a string and return it
func (m *Message) GetString(key string) (string, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey(m) == key {
			if attr.Sval == nil {
				return "", false
			}

			return *attr.Sval, true
		}
	}

	return "", false
}

// Find an Attibute by name that is a byte slice and return it
func (m *Message) GetBytes(key string) ([]byte, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey(m) == key {
			if attr.Bval == nil {
				return nil, false
			}

			return attr.Bval, true
		}
	}

	return nil, false
}

// Find an Attibute by name that is an Interval and return it
func (m *Message) GetInterval(key string) (*Interval, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey(m) == key {
			if attr.Tval == nil {
				return nil, false
			}

			return attr.Tval, true
		}
	}

	return nil, false
}

// Find an Attibute by name that is a boolean and return it
func (m *Message) GetBool(key string) (bool, bool) {
	for _, attr := range m.Attributes {
		if attr.StringKey(m) == key {
			if attr.Boolval == nil {
				return false, false
			}

			return *attr.Boolval, true
		}
	}

	return false, false
}

// Set the SessionID of a Message
func (m *Message) For(id string) {
	m.SessionId = &id
}

// Error indicating that the value type and the attribute type mismatch
var ErrBadValue = errors.New("Invalid type for attribute value")

// Used to convert a value into an int64
type Inter interface {
	Int() int64
}

// Used to convert a valeu into a string
type Stringer interface {
	String() string
}

// Add a tag to the Message
func (m *Message) AddTag(key string, val string) {
	var tag *Tag

	for _, t := range m.Tags {
		if t.Name == key {
			tag = t
			break
		}
	}

	if tag == nil {
		tag = &Tag{Name: key}
		m.Tags = append(m.Tags, tag)
	}

	if val == "" {
		tag.Value = nil
	} else {
		tag.Value = &val
	}
}

// Return the value of a tag
func (m *Message) GetTag(key string) (string, bool) {
	for _, t := range m.Tags {
		if t.Name == key {
			if t.Value == nil {
				return "", true
			}

			return *t.Value, true
		}
	}

	return "", false
}

// Add a new attribute to the Message. The type of the attribute is infered
// from the type of val.
// This understands:
//    int, int32, uint32, int64, uint64, Inter
//    float32, float64,
//    string, Stringer
//    time.Duration
//    []byte
//    error
//    A slice, array, map, or struct containing any understood type
func (m *Message) Add(key string, val interface{}) error {
	attr := &Attribute{}

	if val, ok := versionSymbols[m.Version].FindIndex(key); ok {
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

// Add many attributes to the Message. vals is pairs of (key, value)
// For example:
//    m.AddMany("name", "evan", "age", 35)
// This creates an attributed with a key of "name" and a value of "evan"
// and "age" with a value of 35.
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

// Set the key (ie, the name) of an Attribute.
func (attr *Attribute) SetKey(m *Message, key string) {
	if val, ok := versionSymbols[m.Version].FindIndex(key); ok {
		attr.Key = val
	} else {
		attr.Skey = &key
	}
}

// Add an Attribute of type int64
func (m *Message) AddInt(key string, val int64) error {
	attr := &Attribute{}

	attr.SetKey(m, key)
	attr.Ival = &val

	m.Attributes = append(m.Attributes, attr)
	return nil
}

// Add an Attribute of type float64
func (m *Message) AddFloat(key string, val float64) error {
	attr := &Attribute{}

	attr.SetKey(m, key)
	attr.Fval = &val

	m.Attributes = append(m.Attributes, attr)
	return nil
}

// Add an Attribute of type string
func (m *Message) AddString(key string, val string) error {
	attr := &Attribute{}

	attr.SetKey(m, key)
	attr.Sval = &val

	m.Attributes = append(m.Attributes, attr)
	return nil
}

// Add an Attribute of type bytes
func (m *Message) AddBytes(key string, val []byte) error {
	attr := &Attribute{}

	attr.SetKey(m, key)
	attr.Bval = val

	m.Attributes = append(m.Attributes, attr)
	return nil
}

// Add an Attribute of type Internal
func (m *Message) AddInterval(key string, sec uint64, nsec uint32) error {
	attr := &Attribute{}

	attr.SetKey(m, key)
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

// Add an Attribute of type Internal from a time.Duration
func (m *Message) AddDuration(key string, dur time.Duration) error {
	attr := &Attribute{}

	attr.SetKey(m, key)
	attr.Tval = durationToInterval(dur)

	m.Attributes = append(m.Attributes, attr)
	return nil
}
