package cypress

import "fmt"

type symbols struct {
	StrToIndex map[string]uint32
	IndexToStr []string
}

func newSymbols() *symbols {
	return &symbols{
		make(map[string]uint32),
		[]string{"-"}, // we start the symbol values at 1 so we reserve 0
	}
}

func (s *symbols) Append(strings ...string) {
	for _, str := range strings {
		if _, ok := s.StrToIndex[str]; ok {
			panic(fmt.Sprintf("already assigned %s", str))
		}

		idx := len(s.IndexToStr)

		s.IndexToStr = append(s.IndexToStr, str)
		s.StrToIndex[str] = uint32(idx)
	}
}

func (s *symbols) FromIndex(i uint32) string {
	if i > uint32(len(s.IndexToStr)) {
		return fmt.Sprintf("symbol%d", i)
	}

	return s.IndexToStr[i]
}

func (s *symbols) FindIndex(str string) (uint32, bool) {
	if idx, ok := s.StrToIndex[str]; ok {
		return idx, true
	}

	return 0, false
}

var versionSymbols []*symbols

func init() {
	// do not reorder this function. Do not alphabetize these. They're order
	// in the function dictates their idx. Change them breaks shit.

	v1 := newSymbols()

	v1.Append("message", "value", "source")
	v1.Append("host", "facility", "severity", "tag", "pid")
	v1.Append("name", "type")

	versionSymbols = []*symbols{v1, v1}
}
