package elasticsearch

import "github.com/stretchr/testify/mock"

import "net/http"

type MockConnection struct {
	mock.Mock
}

func (m *MockConnection) Do(_a0 *http.Request) (*http.Response, error) {
	ret := m.Called(_a0)

	r0 := ret.Get(0).(*http.Response)
	r1 := ret.Error(1)

	return r0, r1
}
