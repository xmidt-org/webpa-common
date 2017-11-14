package fanout

import "github.com/stretchr/testify/mock"

type mockRequest struct {
	mock.Mock
}

func (m *mockRequest) Entity() interface{} {
	return m.Called().Get(0)
}
