package cypress

// A slices of Messages that can be used to order messages by Timestamp
type Messages []*Message

func (m Messages) Len() int      { return len(m) }
func (m Messages) Swap(i, j int) { m[i], m[j] = m[j], m[i] }

func (m Messages) Less(i, j int) bool {
	return m[i].GetTimestamp().Before(m[j].GetTimestamp())
}
