package cypress

import "github.com/stretchr/testify/mock"

type MockReceiver struct {
	mock.Mock
}

func (m *MockReceiver) Receive(msg *Message) error {
	ret := m.Called(msg)

	r0 := ret.Error(0)

	return r0
}
func (m *MockReceiver) Close() error {
	ret := m.Called()

	r0 := ret.Error(0)

	return r0
}
