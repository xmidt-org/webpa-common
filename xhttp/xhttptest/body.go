package xhttptest

import "github.com/stretchr/testify/mock"

// MockBody is a stretchr mock for a Request or a Response body, which is really just an io.ReadCloser.
// This is mainly useful when testing error cases.  For testing with actual byte contents, it's generally more convenient
// to use a *bytes.Buffer or other concrete container of bytes instead of mocking.
type MockBody struct {
	mock.Mock
}

// OnReadError sets an expectation for a call to Read, with any byte slice, that returns the given error (and 0 for bytes read).
// If the given error is nil, wierd behavior can occur as the mocked Read will return (0, nil).
func (mb *MockBody) OnReadError(err error) *mock.Call {
	return mb.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, err)
}

// OnCloseError sets an expectation for a call to Close that simply returns the given error.
// The given error can be nil.
func (mb *MockBody) OnCloseError(err error) *mock.Call {
	return mb.On("Close").Return(err)
}

func (mb *MockBody) Read(p []byte) (int, error) {
	arguments := mb.Called(p)
	return arguments.Int(0), arguments.Error(1)
}

func (mb *MockBody) Close() error {
	return mb.Called().Error(0)
}
