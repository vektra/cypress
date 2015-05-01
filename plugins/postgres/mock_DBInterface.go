package postgres

import "github.com/stretchr/testify/mock"

import "database/sql"

import _ "github.com/lib/pq"

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
func (m *MockDBInterface) Query(query string, args ...interface{}) (*sql.Rows, error) {
	ret := m.Called(query, args)

	r0 := ret.Get(0).(*sql.Rows)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *MockDBInterface) Close() error {
	ret := m.Called()

	r0 := ret.Error(0)

	return r0
}
