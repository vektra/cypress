package postgres

import "github.com/stretchr/testify/mock"

import "database/sql"

type MockDBInterface struct {
	mock.Mock
}

func (m *MockDBInterface) Ping() error {
	ret := m.Called()

	r0 := ret.Error(0)

	return r0
}
func (m *MockDBInterface) Exec(query string, args ...interface{}) (sql.Result, error) {
	ret := m.Called(query, args)

	r0 := ret.Get(0).(sql.Result)
	r1 := ret.Error(1)

	return r0, r1
}
