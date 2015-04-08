package cypress

// A type which holds a set of Messages and returns them when requested
// Mostly used for testing.
type StaticGeneratorMessages struct {
	m []*Message
}

// Create a StaticGeneratorMessages from the given Message set
func StaticGenerator(m ...*Message) *StaticGeneratorMessages {
	return &StaticGeneratorMessages{m}
}

// Return the next Message or nil if empty
func (s *StaticGeneratorMessages) Generate() (*Message, error) {
	if len(s.m) == 0 {
		return nil, nil
	}

	m := s.m[0]
	s.m = s.m[1:]

	return m, nil
}

// To satisfy the Generator interface
func (s *StaticGeneratorMessages) Close() error {
	return nil
}
