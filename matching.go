package cypress

type Criteria []map[string]interface{}

func (crit Criteria) Matches(m *Message) bool {
	if len(crit) == 0 {
		return true
	}

	for _, o := range crit {
		matched := true

		for k, v := range o {
			val, ok := m.Get(k)
			if !ok {
				matched = false
				break
			}

			if val != v {
				matched = false
				break
			}
		}

		if matched {
			return true
		}
	}

	return false
}
