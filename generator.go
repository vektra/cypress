package cypress

type StaticGeneratorMessages struct {
	m []*Message
}

func StaticGenerator(m ...*Message) *StaticGeneratorMessages {
	return &StaticGeneratorMessages{m}
}

func (s *StaticGeneratorMessages) Generate() (*Message, error) {
	if len(s.m) == 0 {
		return nil, nil
	}

	m := s.m[0]
	s.m = s.m[1:]

	return m, nil
}

func (s *StaticGeneratorMessages) Close() error {
	return nil
}
