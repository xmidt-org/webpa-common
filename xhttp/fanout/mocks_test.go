package fanout

import (
	"context"
	"net/http"
	"net/url"

	"github.com/stretchr/testify/mock"
)

type mockBody struct {
	mock.Mock
}

func (m *mockBody) Read(p []byte) (int, error) {
	arguments := m.Called(p)
	return arguments.Int(0), arguments.Error(1)
}

func (m *mockBody) Close() error {
	return m.Called().Error(0)
}

type mockErrorEncoder struct {
	mock.Mock
}

func (m *mockErrorEncoder) Encode(ctx context.Context, err error, response http.ResponseWriter) {
	m.Called(ctx, err, response)
}

type mockEndpoints struct {
	mock.Mock
}

func (m *mockEndpoints) NewEndpoints(original *http.Request) ([]*url.URL, error) {
	arguments := m.Called(original)
	first, _ := arguments.Get(0).([]*url.URL)
	return first, arguments.Error(1)
}
