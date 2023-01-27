package tracing

import "github.com/stretchr/testify/mock"

type mockSpanned struct {
	mock.Mock
}

func (m *mockSpanned) Spans() []Span {
	// nolint: typecheck
	return m.Called().Get(0).([]Span)
}

type mockMergeable struct {
	mock.Mock
}

func (m *mockMergeable) Spans() []Span {
	// nolint: typecheck
	return m.Called().Get(0).([]Span)
}

func (m *mockMergeable) WithSpans(spans ...Span) interface{} {
	// nolint: typecheck
	return m.Called(spans).Get(0)
}
