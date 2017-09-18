package wrphttp

import (
	"io"

	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrpendpoint"
	"github.com/go-kit/kit/log"
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

type mockRequestResponse struct {
	mock.Mock
}

func (m *mockRequestResponse) Destination() string {
	return m.Called().String(0)
}

func (m *mockRequestResponse) TransactionID() string {
	return m.Called().String(0)
}

func (m *mockRequestResponse) Message() *wrp.Message {
	return m.Called().Get(0).(*wrp.Message)
}

func (m *mockRequestResponse) Encode(output io.Writer, pool *wrp.EncoderPool) error {
	return m.Called(output, pool).Error(0)
}

func (m *mockRequestResponse) EncodeBytes(pool *wrp.EncoderPool) ([]byte, error) {
	arguments := m.Called(pool)
	return arguments.Get(0).([]byte), arguments.Error(1)
}

func (m *mockRequestResponse) Logger() log.Logger {
	return m.Called().Get(0).(log.Logger)
}

func (m *mockRequestResponse) WithLogger(logger log.Logger) wrpendpoint.Request {
	return m.Called(logger).Get(0).(wrpendpoint.Request)
}

func (m *mockRequestResponse) Spans() []tracing.Span {
	return m.Called().Get(0).([]tracing.Span)
}

func (m *mockRequestResponse) WithSpans(spans ...tracing.Span) interface{} {
	return m.Called(spans).Get(0)
}
