package cypress

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/vektra/tai64n"
)

// Generate JSON for a Message
func (m *Message) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.SimpleJSONMap())
}

// Populate a Message from JSON in data
func (m *Message) UnmarshalJSON(data []byte) error {
	m2, err := ParseSimpleJSON(data)
	if err != nil {
		return err
	}

	*m = *m2

	return err
}

// Generate a Message from json in data
func ParseSimpleJSON(data []byte) (*Message, error) {
	m := &Message{Version: DEFAULT_VERSION}

	var p map[string]interface{}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	err := dec.Decode(&p)
	if err != nil {
		return nil, err
	}

	for key, val := range p {
		switch key {
		case "@version":
			// skip
		case "@type":
			if str, ok := val.(string); ok {
				switch str {
				case "metric":
					m.Type = proto.Uint32(METRIC)
				case "trace":
					m.Type = proto.Uint32(TRACE)
				case "audit":
					m.Type = proto.Uint32(AUDIT)
				case "heartbeat":
					m.Type = proto.Uint32(HEARTBEAT)
				}
			}

		case "@timestamp":
			if str, ok := val.(string); ok {
				ts, err := time.Parse(time.RFC3339Nano, str)
				if err != nil {
					return nil, err
				}

				m.Timestamp = tai64n.FromTime(ts)
			}
		case "@tags":
			if tm, ok := val.(map[string]interface{}); ok {
				for tkey, tiv := range tm {
					tval, vok := tiv.(string)

					if vok {
						if tval == "true" {
							m.Tags = append(m.Tags, &Tag{Name: tkey})
						} else {
							m.Tags = append(m.Tags, &Tag{Name: tkey, Value: &tval})
						}
					}
				}
			}
		default:
			if num, ok := val.(json.Number); ok {
				i, err := num.Int64()
				if err == nil {
					m.AddInt(key, i)
				} else {
					f, err := num.Float64()
					if err != nil {
						return nil, err
					}

					m.AddFloat(key, f)
				}
			} else {
				if imap, ok := val.(map[string]interface{}); ok {
					fsec, sok := imap["seconds"].(json.Number)
					nsec, nok := imap["nanoseconds"].(json.Number)

					if sok && nok {
						pfsec, err := fsec.Int64()
						if err != nil {
							return nil, ErrInvalidMessage
						}

						pnsec, err := nsec.Int64()
						if err != nil {
							return nil, ErrInvalidMessage
						}

						m.AddInterval(key, uint64(pfsec), uint32(pnsec))
					} else {
						sval, ok := imap["bytes"].(string)
						if ok {
							bytes, err := base64.StdEncoding.DecodeString(sval)
							if err != nil {
								return nil, ErrInvalidMessage
							}

							m.AddBytes(key, bytes)
						} else {
							err = m.Add(key, val)
							if err != nil {
								return nil, err
							}
						}
					}
				} else {
					m.Add(key, val)
				}
			}
		}
	}

	if m.Type == nil {
		m.Type = proto.Uint32(LOG)
	}

	if m.Timestamp == nil {
		m.Timestamp = tai64n.Now()
	}

	return m, nil
}

func findTarget(key string, m map[string]interface{}) (string, map[string]interface{}) {
	parts := strings.Split(key, ".")

	for _, part := range parts[:len(parts)-1] {
		s, ok := m[part]
		if !ok {
			sm := make(map[string]interface{})
			m[part] = sm
			m = sm
		} else {
			sm, ok := s.(map[string]interface{})
			if ok {
				m = sm
			} else {
				break
			}
		}
	}

	return parts[len(parts)-1], m
}

// Return a simple map representing the Message used to generate
// JSON
func (m *Message) SimpleJSONMap() map[string]interface{} {
	p := map[string]interface{}{
		"@timestamp": m.Timestamp.Time().Format(time.RFC3339Nano),
		"@type":      m.StringType(),
		"@version":   "1", // make this compatible with logstash
	}

	if len(m.Tags) > 0 {
		tags := map[string]string{}

		for _, tag := range m.Tags {
			var val string

			if tag.Value == nil {
				val = "true"
			} else {
				val = tag.GetValue()
			}

			tags[tag.Name] = val
		}

		p["@tags"] = tags
	}

	for _, attr := range m.Attributes {
		var val interface{}

		key := attr.StringKey(m)

		var target map[string]interface{}

		if strings.Contains(key, ".") {
			key, target = findTarget(key, p)
		} else {
			target = p
		}

		switch {
		case attr.Ival != nil:
			val = *attr.Ival
		case attr.Fval != nil:
			val = *attr.Fval
		case attr.Boolval != nil:
			val = *attr.Boolval
		case attr.Sval != nil:
			val = *attr.Sval
		case attr.Bval != nil:
			val = map[string][]byte{
				"bytes": attr.Bval,
			}
		case attr.Tval != nil:
			val = map[string]uint64{
				"seconds":     attr.Tval.GetSeconds(),
				"nanoseconds": uint64(attr.Tval.GetNanoseconds()),
			}
		default:
			val = true
		}

		target[key] = val
	}

	return p
}
