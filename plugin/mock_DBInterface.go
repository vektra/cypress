package plugin

import "github.com/stretchr/testify/mock"

type MockDBInterface struct {
	mock.Mock
}

func (m *MockDBInterface) Ping() error {
	ret := m.Called()

	r0 := ret.Error(0)

	return r0
}
func (m *MockDBInterface) Exec(_a0 string) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return r0
}
func (m *MockDBInterface) Prepare(_a0 string) (StmtInterface, error) {
	ret := m.Called(_a0)

	r0 := ret.Get(0).(StmtInterface)
	r1 := ret.Error(1)

	return r0, r1
}
