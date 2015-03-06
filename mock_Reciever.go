package cypress

import "github.com/stretchr/testify/mock"

type MockReceiver struct {
	mock.Mock
}

func (m *MockReceiver) Read(msg *Message) error {
	ret := m.Called(msg)

	r0 := ret.Error(0)

	return r0
}
