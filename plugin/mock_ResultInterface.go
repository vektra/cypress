package plugin

import "github.com/stretchr/testify/mock"

type MockResultInterface struct {
	mock.Mock
}

func (m *MockResultInterface) LastInsertId() (int64, error) {
	ret := m.Called()

	r0 := ret.Get(0).(int64)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *MockResultInterface) RowsAffected() (int64, error) {
	ret := m.Called()

	r0 := ret.Get(0).(int64)
	r1 := ret.Error(1)

	return r0, r1
}
