package wrphttp

import (
	"io"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/mock"
)

type mockReadCloser struct {
	mock.Mock
}

func (m *mockReadCloser) Read(p []byte) (int, error) {
	arguments := m.Called(p)
	return arguments.Int(0), arguments.Error(1)
}

func (m *mockReadCloser) Close() error {
	return m.Called().Error(0)
}

type mockResponse struct {
	mock.Mock
}

func (m *mockResponse) Destination() string {
	return m.Called().String(0)
}

func (m *mockResponse) TransactionID() string {
	return m.Called().String(0)
}

func (m *mockResponse) Message() *wrp.Message {
	return m.Called().Get(0).(*wrp.Message)
}

func (m *mockResponse) Encode(output io.Writer, pool *wrp.EncoderPool) error {
	return m.Called(output, pool).Error(0)
}

func (m *mockResponse) EncodeBytes(pool *wrp.EncoderPool) ([]byte, error) {
	arguments := m.Called(pool)
	return arguments.Get(0).([]byte), arguments.Error(1)
}
