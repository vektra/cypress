package plugin

import "github.com/stretchr/testify/mock"

type MockStmtInterface struct {
	mock.Mock
}

func (m *MockStmtInterface) Exec(args ...interface{}) (interface{}, error) {
	ret := m.Called(args)

	r0 := ret.Get(0).(interface{})
	r1 := ret.Error(1)

	return r0, r1
}
