package tracing

import "github.com/stretchr/testify/mock"

type mockSpanned struct {
	mock.Mock
}

func (m *mockSpanned) Spans() []Span {
	return m.Called().Get(0).([]Span)
}

type mockMergeable struct {
	mock.Mock
}

func (m *mockMergeable) Spans() []Span {
	return m.Called().Get(0).([]Span)
}

func (m *mockMergeable) WithSpans(spans ...Span) interface{} {
	return m.Called(spans).Get(0)
}
