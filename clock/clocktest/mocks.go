package clocktest

import (
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/webpa-common/clock"
)

// Mock is a stretchr mock for a clock.  In addition to implementing clock.Interface and supplying
// mock behavior, other methods that make mocking a bit easier are supplied.
type Mock struct {
	mock.Mock
}

var _ clock.Interface = (*Mock)(nil)

func (m *Mock) Now() time.Time {
	return m.Called().Get(0).(time.Time)
}

func (m *Mock) OnNow(v time.Time) *mock.Call {
	return m.On("Now").Return(v)
}

func (m *Mock) Sleep(d time.Duration) {
	m.Called(d)
}

func (m *Mock) OnSleep(d time.Duration) *mock.Call {
	return m.On("Sleep", d)
}

func (m *Mock) NewTimer(d time.Duration) clock.Timer {
	return m.Called(d).Get(0).(clock.Timer)
}

func (m *Mock) OnNewTimer(d time.Duration, t clock.Timer) *mock.Call {
	return m.On("NewTimer", d).Return(t)
}

func (m *Mock) NewTicker(d time.Duration) clock.Ticker {
	return m.Called(d).Get(0).(clock.Ticker)
}

func (m *Mock) OnNewTicker(d time.Duration, t clock.Ticker) *mock.Call {
	return m.On("NewTicker", d).Return(t)
}

// MockTimer is a stretchr mock for the clock.Timer interface
type MockTimer struct {
	mock.Mock
}

var _ clock.Timer = (*MockTimer)(nil)

func (m *MockTimer) C() <-chan time.Time {
	return m.Called().Get(0).(<-chan time.Time)
}

func (m *MockTimer) OnC(c <-chan time.Time) *mock.Call {
	return m.On("C").Return(c)
}

func (m *MockTimer) Reset(d time.Duration) bool {
	return m.Called(d).Bool(0)
}

func (m *MockTimer) OnReset(d time.Duration, r bool) *mock.Call {
	return m.On("Reset", d).Return(r)
}

func (m *MockTimer) Stop() bool {
	return m.Called().Bool(0)
}

func (m *MockTimer) OnStop(r bool) *mock.Call {
	return m.On("Stop").Return(r)
}

type MockTicker struct {
	mock.Mock
}

var _ clock.Ticker = (*MockTicker)(nil)

func (m *MockTicker) C() <-chan time.Time {
	return m.Called().Get(0).(<-chan time.Time)
}

func (m *MockTicker) OnC(c <-chan time.Time) *mock.Call {
	return m.On("C").Return(c)
}

func (m *MockTicker) Stop() {
	m.Called()
}

func (m *MockTicker) OnStop() *mock.Call {
	return m.On("Stop")
}
