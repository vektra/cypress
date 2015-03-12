package cypress

import (
	"bytes"
	"encoding/json"
	"testing"

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

func TestJsonBytes(t *testing.T) {
	m := Log()
	err := m.Add("message", []byte("This is a test"))
	err = m.Add("blah", "foo")
	err = m.Add("add", 10)

	if err != nil {
		panic(err)
	}

	b, err := json.Marshal(m)

	if err != nil {
		panic(err)
	}

	var om Message

	err = json.Unmarshal(b, &om)

	if err != nil {
		panic(err)
	}

	b2, err := json.Marshal(om)

	if !bytes.Equal(b, b2) {
		t.Errorf("Roundtrip through json failed: '%s' != '%s'", string(b), string(b2))
	}
}

func TestJsonInterval(t *testing.T) {
	m := Log()
	err := m.AddInterval("time", 10, 2)

	if err != nil {
		panic(err)
	}

	b, err := json.Marshal(m)

	if err != nil {
		panic(err)
	}

	var om Message

	err = json.Unmarshal(b, &om)

	if err != nil {
		panic(err)
	}

	v, ok := om.GetInterval("time")

	if !ok {
		t.Errorf("Unable to find time")
	} else if v.GetSeconds() != 10 || v.GetNanoseconds() != 2 {
		t.Errorf("time didn't roundtrip")
	}

	b2, err := json.Marshal(om)

	if !bytes.Equal(b, b2) {
		t.Errorf("Roundtrip through json failed: '%s' != '%s'", string(b), string(b2))
	}
}

func checkInt(t *testing.T, m *Message, key string, exp int64) {
	act, ok := m.GetInt(key)

	if !ok {
		t.Errorf("Didn't parse %s correctly", key)
	}

	if exp != act {
		t.Errorf("Didn't parse %s correctly", key)
	}
}

func checkFloat(t *testing.T, m *Message, key string, exp float64) {
	act, ok := m.GetFloat(key)

	if !ok {
		t.Errorf("Didn't parse %s correctly", key)
	}

	if exp != act {
		t.Errorf("Didn't parse %s correctly", key)
	}
}

func checkStr(t *testing.T, m *Message, key string, exp string) {
	act, ok := m.GetString(key)

	if !ok {
		t.Errorf("Didn't parse %s correctly", key)
	}

	if exp != act {
		t.Errorf("Didn't parse %s correctly", key)
	}
}

func TestParseKV(t *testing.T) {
	line := `> id=1 remote="10.1.42.1" time="1377377621.034" method=GET request="/test" size=171 proc=3.251 status=200 body=157 for="-"`

	m, err := ParseKV(line)

	if err != nil {
		t.Fatalf("Unable to parse line")
	}

	checkInt(t, m, "id", 1)
	checkInt(t, m, "size", 171)
	checkStr(t, m, "for", "-")
	checkFloat(t, m, "proc", 3.251)
}

func TestParseKVSeconds(t *testing.T) {
	line := `> id=1 remote="10.1.42.1" time=:1377377621.034 method=GET request="/test" size=171 proc=:0.000 status=200 body=157 for="-"`

	m, err := ParseKV(line)

	if err != nil {
		t.Fatalf("Unable to parse line")
	}

	checkInt(t, m, "id", 1)
	checkInt(t, m, "size", 171)
	checkStr(t, m, "for", "-")

	act, ok := m.GetInterval("time")

	if !ok {
		t.Errorf("Didn't parse time correctly")
	}

	if act.GetSeconds() != 1377377621 || act.GetNanoseconds() != 34000000 {
		t.Errorf("Didn't parse time correctly %d %d", act.GetSeconds(), act.GetNanoseconds())
	}
}

func TestParseKVStream(t *testing.T) {
	line := "> id=1 size=171 for=\"-\"\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 2 {
		t.Errorf("Didn't parse 2 message: %d", len(mbuf.Messages))
	}

	for _, m := range mbuf.Messages {
		checkInt(t, m, "id", 1)
		checkInt(t, m, "size", 171)
		checkStr(t, m, "for", "-")
	}
}

func TestParseKVStreamSkipNonLogLine(t *testing.T) {
	line := "> id=1 size=171 for=\"-\"\nblah blah\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 2 {
		t.Errorf("Didn't parse 2 message")
	}

	for _, m := range mbuf.Messages {
		checkInt(t, m, "id", 1)
		checkInt(t, m, "size", 171)
		checkStr(t, m, "for", "-")
	}
}

func TestParseKVStreamAfterBlankLine(t *testing.T) {
	line := "\n> id=1 size=171 for=\"-\"\nblah blah\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 2 {
		t.Errorf("Didn't parse 2 message: %d", len(mbuf.Messages))
	}

	for _, m := range mbuf.Messages {
		checkInt(t, m, "id", 1)
		checkInt(t, m, "size", 171)
		checkStr(t, m, "for", "-")
	}
}

func TestParseKVStreamAfterJunkAndBlankLine(t *testing.T) {
	line := "blahblah\n\n> id=1 size=171 for=\"-\"\nblah blah\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 2 {
		t.Errorf("Didn't parse 2 message: %d", len(mbuf.Messages))
	}

	for _, m := range mbuf.Messages {
		checkInt(t, m, "id", 1)
		checkInt(t, m, "size", 171)
		checkStr(t, m, "for", "-")
	}
}

func TestParseKVStreamBadLogLine(t *testing.T) {
	line := "> id=1 size=171 for=\"-\"\n> id=2 blah blah\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 2 {
		t.Errorf("Didn't parse 2 message")
	}

	for _, m := range mbuf.Messages {
		if m.GetType() != tLog {
			t.Errorf("Type not a metric")
		}

		checkInt(t, m, "id", 1)
		checkInt(t, m, "size", 171)
		checkStr(t, m, "for", "-")
	}
}

func TestParseKVStreamMetric(t *testing.T) {
	line := ">! id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 1 {
		t.Fatalf("Didn't parse 1 message: %d", len(mbuf.Messages))
	}

	m := mbuf.Messages[0]

	if m.GetType() != tMetric {
		t.Errorf("Type not a metric")
	}

	checkInt(t, m, "id", 1)
	checkInt(t, m, "size", 171)
	checkStr(t, m, "for", "-")
}

func TestParseKVStreamTrace(t *testing.T) {
	line := ">$ id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 1 {
		t.Fatalf("Didn't parse 1 message: %d", len(mbuf.Messages))
	}

	m := mbuf.Messages[0]

	if m.GetType() != tTrace {
		t.Errorf("Type not a trace")
	}

	checkInt(t, m, "id", 1)
	checkInt(t, m, "size", 171)
	checkStr(t, m, "for", "-")
}

func TestParseKVStreamAudit(t *testing.T) {
	line := ">* id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 1 {
		t.Fatalf("Didn't parse 1 message: %d", len(mbuf.Messages))
	}

	m := mbuf.Messages[0]

	if m.GetType() != tAudit {
		t.Errorf("Type not a trace")
	}

	checkInt(t, m, "id", 1)
	checkInt(t, m, "size", 171)
	checkStr(t, m, "for", "-")
}

func TestParseKVStreamWithBare(t *testing.T) {
	line := "> id=1 size=171 for=\"-\"\nerror bad stuff\n> id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	s := KVStream{buf, &mbuf, true, ""}

	s.Parse()

	if len(mbuf.Messages) != 3 {
		t.Fatalf("Didn't parse 2 message: %d", len(mbuf.Messages))
	}

	m := mbuf.Messages[1]

	checkStr(t, m, "message", "error bad stuff")
}

func TestParseKVStreamPreStamped(t *testing.T) {
	line := "> @40000000521BBC7D163ED116 id=1 size=171 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 1 {
		t.Errorf("Didn't parse 2 message: %d", len(mbuf.Messages))
	}

	m := mbuf.Messages[0]

	if m.GetTimestamp().Label() != "@40000000521BBC7D163ED116" {
		t.Errorf("Timestamp didn't parse correctly")
	}
}

func TestMessageKVString(t *testing.T) {
	line := "> @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 1 {
		t.Errorf("Didn't parse 2 message: %d", len(mbuf.Messages))
	}

	m := mbuf.Messages[0]

	kv := m.KVString()

	if kv != "> @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\"" {
		t.Errorf("KVString didn't output proper format: '%s'", kv)
	}
}

func TestMessageKVStringTimestamp(t *testing.T) {
	line := "> id=1 time=:1.001 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 1 {
		t.Errorf("Didn't parse 2 message: %d", len(mbuf.Messages))
	}

	m := mbuf.Messages[0]

	kv := m.KVString()

	exc := "> " + m.GetTimestamp().Label() + " id=1 time=:1.001 for=\"-\""

	if kv != exc {
		t.Errorf("KVString didn't output proper format: '%s'", kv)
	}
}

func TestMessageKVStringMetric(t *testing.T) {
	line := ">! @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 1 {
		t.Errorf("Didn't parse 2 message: %d", len(mbuf.Messages))
	}

	m := mbuf.Messages[0]

	kv := m.KVString()

	if kv != ">! @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\"" {
		t.Errorf("KVString didn't output proper format: '%s'", kv)
	}
}

func TestMessageKVTrace(t *testing.T) {
	line := ">$ @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\""

	buf := bytes.NewReader([]byte(line))
	var mbuf MessageBuffer

	ParseKVStream(buf, &mbuf)

	if len(mbuf.Messages) != 1 {
		t.Errorf("Didn't parse 2 message: %d", len(mbuf.Messages))
	}

	m := mbuf.Messages[0]

	kv := m.KVString()

	if kv != ">$ @40000000521BBC7D163ED116 id=1 time=:1.001 for=\"-\"" {
		t.Errorf("KVString didn't output proper format: '%s'", kv)
	}
}

func TestAddSlice(t *testing.T) {
	m := Log()
	err := m.Add("things", []string{"one", "two", "three"})
	require.NoError(t, err)

	data, err := ToProtobuf(m)
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

	data, err := ToProtobuf(m)
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

	data, err := ToProtobuf(m)
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

	data, err := ToProtobuf(m)
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

	data, err := ToProtobuf(m)
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
