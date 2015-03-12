package cypress

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func (attr Attribute) MarshalJSON() ([]byte, error) {
	var key string

	key = attr.StringKey()

	switch {
	case attr.Ival != nil:
		return []byte(fmt.Sprintf("{ \"%s\": %d }", key, *attr.Ival)), nil
	case attr.Fval != nil:
		return []byte(fmt.Sprintf("{ \"%s\": %f }", key, *attr.Fval)), nil
	case attr.Sval != nil:
		v, err := json.Marshal(attr.Sval)
		if err != nil {
			return nil, err
		}

		return []byte(fmt.Sprintf("{ \"%s\": %s }", key, v)), nil
	case attr.Bval != nil:
		v, err := json.Marshal(attr.Bval)
		if err != nil {
			return nil, err
		}

		// The value for _bytes would naturally be true, but we want to keep
		// the literal as a pure map[string]string so we use an empty string.
		// It's just the presence of _bytes that matters anyway
		return []byte(fmt.Sprintf("{ \"%s\": %s, \"_bytes\": \"\" }", key, string(v))), nil
	case attr.Tval != nil:
		v, err := json.Marshal(attr.Tval)

		if err != nil {
			return nil, err
		}

		return []byte(fmt.Sprintf("{ \"%s\": %s }", key, v)), nil
	default:
		return []byte(fmt.Sprintf("{ \"%s\": 1 }", key)), nil
	}
}

func (attr *Attribute) UnmarshalJSON(data []byte) error {
	// Order is arbitrary. We don't parse as maps to interfaces to avoid floaty ints.
	parsers := [](func([]byte) (*Attribute, error)){
		parseIntMap,
		parseStringMap,
		parseIntervalMap,
	}

	// TODO(kev): Probably not worth it, but if parsing becomes a bottleneck
	//            we can toss these into goroutines and just grab the first one
	//            that comes back without error
	for _, parser := range parsers {
		if parsed, err := parser(data); err == nil {
			*attr = *parsed
			return nil
		}
	}

	return errors.New("Unable to parse json")
}

func parseIntMap(data []byte) (*Attribute, error) {
	var intMap map[string]int64

	err := json.Unmarshal(data, &intMap)

	if err != nil {
		return nil, err
	}

	var out Attribute

	for k, v := range intMap {
		if idx, ok := PresetKeys[k]; ok {
			out.Key = idx
		} else {
			out.Skey = &k
		}

		out.Ival = &v
		break
	}

	return &out, nil
}

func parseStringMap(data []byte) (*Attribute, error) {
	var raw map[string]string

	err := json.Unmarshal(data, &raw)

	if err != nil {
		return nil, err
	}

	var out Attribute

	for k, v := range raw {
		if k == "_bytes" {
			continue
		}

		if idx, ok := PresetKeys[k]; ok {
			out.Key = idx
		} else {
			out.Skey = &k
		}

		_, bytes := raw["_bytes"]

		if bytes {
			b, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return nil, err
			}
			out.Bval = b
		} else {
			out.Sval = &v
		}
		break
	}

	return &out, nil
}

func parseIntervalMap(data []byte) (*Attribute, error) {
	var tvalMap map[string]*Interval

	err := json.Unmarshal(data, &tvalMap)

	if err != nil {
		return nil, err
	}

	var out Attribute

	for k, v := range tvalMap {
		if idx, ok := PresetKeys[k]; ok {
			out.Key = idx
		} else {
			out.Skey = &k
		}

		out.Tval = v
		break
	}

	return &out, nil
}

type JsonStream struct {
	Src io.Reader
	Out Receiver
}

func (js *JsonStream) Parse() error {
	dec := json.NewDecoder(js.Src)

	for {
		m := &Message{}

		err := dec.Decode(m)
		if err != nil {
			return err
		}

		js.Out.Receive(m)
	}

	return nil
}
