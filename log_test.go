package cypress

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddBool(t *testing.T) {
	m := Log()

	err := m.Add("started", true)
	require.NoError(t, err)

	v, ok := m.Get("started")
	require.True(t, ok)

	assert.True(t, v.(bool))
}

func checkInt(t *testing.T, m *Message, key string, exp int64) {
	act, ok := m.GetInt(key)
	require.True(t, ok)
	assert.Equal(t, exp, act)
}

func checkFloat(t *testing.T, m *Message, key string, exp float64) {
	act, ok := m.GetFloat(key)
	require.True(t, ok)
	assert.Equal(t, exp, act)
}

func checkStr(t *testing.T, m *Message, key string, exp string) {
	act, ok := m.GetString(key)
	require.True(t, ok)
	assert.Equal(t, exp, act)
}

func TestParseKV(t *testing.T) {
	line := `> id=1 remote="10.1.42.1" time="1377377621.034" method=GET request="/test" size=171 proc=3.251 status=200 body=157 for="-"`

	m, err := ParseKV(line)
	require.NoError(t, err)

	checkInt(t, m, "id", 1)
	checkInt(t, m, "size", 171)
	checkStr(t, m, "for", "-")
	checkFloat(t, m, "proc", 3.251)
}

func TestParseKVWithStrangeKeys(t *testing.T) {
	line := `> [host.name="foo" "remote.addr"="xyz"] key.id=1 "remote.host"="10.1.42.1"`

	m, err := ParseKV(line)
	require.NoError(t, err)

	assert.Equal(t, "host.name", m.Tags[0].Name)
	assert.Equal(t, "foo", m.Tags[0].GetValue())

	assert.Equal(t, "remote.addr", m.Tags[1].Name)
	assert.Equal(t, "xyz", m.Tags[1].GetValue())

	checkInt(t, m, "key.id", 1)
	checkStr(t, m, "remote.host", "10.1.42.1")
}

func TestParseKVSeconds(t *testing.T) {
	line := `> id=1 remote="10.1.42.1" time=:1377377621.034 method=GET request="/test" size=171 proc=:0.000 status=200 body=157 for="-"`

	m, err := ParseKV(line)
	require.NoError(t, err)

	checkInt(t, m, "id", 1)
	checkInt(t, m, "size", 171)
	checkStr(t, m, "for", "-")

	act, ok := m.GetInterval("time")
	require.True(t, ok)

	assert.Equal(t, uint64(1377377621), act.GetSeconds())
	assert.Equal(t, uint32(34000000), act.GetNanoseconds())
}

func TestParseKVStream(t *testing.T) {
	line := "> id=1 size=171 for=\"-\"\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 2, len(mbuf.Messages))

	for _, m := range mbuf.Messages {
		checkInt(t, m, "id", 1)
		checkInt(t, m, "size", 171)
		checkStr(t, m, "for", "-")
	}
}

func TestParseKVStreamSkipNonLogLine(t *testing.T) {
	line := "> id=1 size=171 for=\"-\"\nblah blah\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 2, len(mbuf.Messages))

	for _, m := range mbuf.Messages {
		checkInt(t, m, "id", 1)
		checkInt(t, m, "size", 171)
		checkStr(t, m, "for", "-")
	}
}

func TestParseKVStreamAfterBlankLine(t *testing.T) {
	line := "\n> id=1 size=171 for=\"-\"\nblah blah\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 2, len(mbuf.Messages))

	for _, m := range mbuf.Messages {
		checkInt(t, m, "id", 1)
		checkInt(t, m, "size", 171)
		checkStr(t, m, "for", "-")
	}
}

func TestParseKVStreamAfterJunkAndBlankLine(t *testing.T) {
	line := "blahblah\n\n> id=1 size=171 for=\"-\"\nblah blah\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 2, len(mbuf.Messages))

	for _, m := range mbuf.Messages {
		checkInt(t, m, "id", 1)
		checkInt(t, m, "size", 171)
		checkStr(t, m, "for", "-")
	}
}

func TestParseKVStreamBadLogLine(t *testing.T) {
	line := "> id=1 size=171 for=\"-\"\n> id=2 blah blah\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 2, len(mbuf.Messages))

	for _, m := range mbuf.Messages {
		checkInt(t, m, "id", 1)
		checkInt(t, m, "size", 171)
		checkStr(t, m, "for", "-")
	}
}

func TestParseKVStreamMetric(t *testing.T) {
	line := ">! id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	assert.Equal(t, uint32(METRIC), m.GetType())

	checkInt(t, m, "id", 1)
	checkInt(t, m, "size", 171)
	checkStr(t, m, "for", "-")
}

func TestParseKVStreamTrace(t *testing.T) {
	line := ">$ id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	assert.Equal(t, uint32(TRACE), m.GetType())

	checkInt(t, m, "id", 1)
	checkInt(t, m, "size", 171)
	checkStr(t, m, "for", "-")
}

func TestParseKVStreamAudit(t *testing.T) {
	line := ">* id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	assert.Equal(t, uint32(AUDIT), m.GetType())

	checkInt(t, m, "id", 1)
	checkInt(t, m, "size", 171)
	checkStr(t, m, "for", "-")
}

func TestParseKVStreamHeartbeat(t *testing.T) {
	line := ">? [host=\"foo\"]"

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	assert.Equal(t, uint32(HEARTBEAT), m.GetType())

	assert.Equal(t, "host", m.Tags[0].Name)
	assert.Equal(t, "foo", m.Tags[0].GetValue())
}

func TestParseKVWithBare(t *testing.T) {
	line := "> id=1 size=171 for=\"-\"\nerror bad stuff\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	parser := NewKVParser(buf)
	parser.Bare = true

	Glue(parser, &mbuf)

	assert.Equal(t, 3, len(mbuf.Messages))

	m := mbuf.Messages[1]

	checkStr(t, m, "message", "error bad stuff")
}

func TestParseKVStreamPreStamped(t *testing.T) {
	line := "> @40000000521BBC7D163ED116 id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	assert.Equal(t, "@40000000521BBC7D163ED116", m.GetTimestamp().Label())
}

func TestMessageKVString(t *testing.T) {
	line := "> @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	kv := m.KVString()

	assert.Equal(t, "> @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\"", kv)
}

func TestMessageKVStringTimestamp(t *testing.T) {
	line := "> id=1 time=:1.001 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	kv := m.KVString()

	exc := "> " + m.GetTimestamp().Label() + " id=1 time=:1.001 for=\"-\""

	assert.Equal(t, exc, kv)
}

func TestMessageKVStringMetric(t *testing.T) {
	line := ">! @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	kv := m.KVString()

	assert.Equal(t, ">! @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\"", kv)
}

func TestMessageKVTrace(t *testing.T) {
	line := ">$ @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	kv := m.KVString()

	assert.Equal(t, ">$ @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\"", kv)
}

func TestMessageKVStringIncludeTags(t *testing.T) {
	line := "> @40000000521BBC7D163ED116 [region=\"us-west-1\"] id=1 time=:1.001 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	kv := m.KVString()

	assert.Equal(t, "> @40000000521BBC7D163ED116 [region=\"us-west-1\"] id=1 time=:1.001 for=\"-\"", kv)
}

func TestMessageKVStringIncludeValuelessTags(t *testing.T) {
	line := "> @40000000521BBC7D163ED116 [region=\"us-west-1\" !secure] id=1 time=:1.001 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf BufferReceiver

	ParseKVStream(buf, &mbuf)

	assert.Equal(t, 1, len(mbuf.Messages))

	m := mbuf.Messages[0]

	kv := m.KVString()

	assert.Equal(t, "> @40000000521BBC7D163ED116 [region=\"us-west-1\" !secure] id=1 time=:1.001 for=\"-\"", kv)
}

func TestAddSlice(t *testing.T) {
	m := Log()
	err := m.Add("things", []string{"one", "two", "three"})
	require.NoError(t, err)

	data, err := m.Marshal()
	require.NoError(t, err)

	m2, err := FromProtobuf(data)
	require.NoError(t, err)

	one, ok := m2.Get("things.0")
	require.True(t, ok)

	assert.Equal(t, one, "one")

	two, ok := m2.Get("things.1")
	require.True(t, ok)

	assert.Equal(t, two, "two")

	three, ok := m2.Get("things.2")
	require.True(t, ok)

	assert.Equal(t, three, "three")
}

func TestAddArray(t *testing.T) {
	m := Log()
	err := m.Add("things", [3]string{"one", "two", "three"})
	require.NoError(t, err)

	data, err := m.Marshal()
	require.NoError(t, err)

	m2, err := FromProtobuf(data)
	require.NoError(t, err)

	one, ok := m2.Get("things.0")
	require.True(t, ok)

	assert.Equal(t, one, "one")

	two, ok := m2.Get("things.1")
	require.True(t, ok)

	assert.Equal(t, two, "two")

	three, ok := m2.Get("things.2")
	require.True(t, ok)

	assert.Equal(t, three, "three")
}

func TestAddMap(t *testing.T) {
	m := Log()
	err := m.Add("things", map[string]int{"one": 1, "two": 2, "three": 3})
	require.NoError(t, err)

	data, err := m.Marshal()
	require.NoError(t, err)

	m2, err := FromProtobuf(data)
	require.NoError(t, err)

	one, ok := m2.Get("things.one")
	require.True(t, ok)

	assert.Equal(t, one, 1)

	two, ok := m2.Get("things.two")
	require.True(t, ok)

	assert.Equal(t, two, 2)

	three, ok := m2.Get("things.three")
	require.True(t, ok)

	assert.Equal(t, three, 3)
}

type simpleThing struct {
	Name string
	Age  int
}

func TestAddStruct(t *testing.T) {
	m := Log()
	err := m.Add("things", simpleThing{Name: "test", Age: 18})
	require.NoError(t, err)

	data, err := m.Marshal()
	require.NoError(t, err)

	m2, err := FromProtobuf(data)
	require.NoError(t, err)

	name, ok := m2.Get("things.name")
	require.True(t, ok)

	assert.Equal(t, name, "test")

	age, ok := m2.Get("things.age")
	require.True(t, ok)

	assert.Equal(t, age, 18)
}

func TestAddPointerToStruct(t *testing.T) {
	m := Log()
	err := m.Add("things", &simpleThing{Name: "test", Age: 18})
	require.NoError(t, err)

	data, err := m.Marshal()
	require.NoError(t, err)

	m2, err := FromProtobuf(data)
	require.NoError(t, err)

	name, ok := m2.Get("things.name")
	require.True(t, ok)

	assert.Equal(t, name, "test")

	age, ok := m2.Get("things.age")
	require.True(t, ok)

	assert.Equal(t, age, 18)
}

func FromProtobuf(buf []byte) (*Message, error) {
	m := &Message{}

	err := proto.Unmarshal(buf, m)
	if err != nil {
		return nil, err
	}

	return m, nil
}
