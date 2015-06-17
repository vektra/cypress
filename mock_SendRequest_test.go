package cypress

import "github.com/stretchr/testify/mock"

type MockSendRequest struct {
	mock.Mock
}

func (m *MockSendRequest) Ack(_a0 *Message) {
	m.Called(_a0)
}
func (m *MockSendRequest) Nack(_a0 *Message) {
	m.Called(_a0)
}
