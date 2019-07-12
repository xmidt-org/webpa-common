package health

import (
	"errors"
	"github.com/xmidt-org/webpa-common/health"
	"github.com/xmidt-org/webpa-common/httppool"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
)

type mockMonitor struct {
	mock.Mock
}

func (monitor *mockMonitor) SendEvent(healthFunc health.HealthFunc) {
	monitor.Called(healthFunc)
}

func (monitor *mockMonitor) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	monitor.Called(response, request)
}

func matchInc(stat health.Stat, delta int) interface{} {
	return mock.MatchedBy(
		func(healthFunc health.HealthFunc) bool {
			stats := make(health.Stats, 1)
			healthFunc(stats)

			return len(stats) == 1 && stats[stat] == 1
		},
	)
}

type mockEvent struct {
	mock.Mock
}

func (event *mockEvent) Type() httppool.EventType {
	arguments := event.Called()
	return arguments.Get(0).(httppool.EventType)
}

func (event *mockEvent) Err() error {
	arguments := event.Called()
	return arguments.Error(0)
}

func TestListener(t *testing.T) {
	var testData = []struct {
		eventType    httppool.EventType
		needsError   bool
		eventError   error
		expectedStat health.Stat
	}{
		{
			eventType:    httppool.EventTypeQueue,
			expectedStat: TotalNotificationsQueued,
		},
		{
			eventType:    httppool.EventTypeReject,
			expectedStat: TotalNotificationsRejected,
		},
		{
			eventType:    httppool.EventTypeFinish,
			needsError:   true,
			expectedStat: TotalNotificationsSucceeded,
		},
		{
			eventType:    httppool.EventTypeFinish,
			eventError:   errors.New("expected"),
			expectedStat: TotalNotificationsFailed,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		mockEvent := &mockEvent{}
		mockEvent.On("Type").Return(record.eventType).Once()

		if record.needsError || record.eventError != nil {
			mockEvent.On("Err").Return(record.eventError).Once()
		}

		mockMonitor := &mockMonitor{}
		mockMonitor.On("SendEvent", matchInc(record.expectedStat, 1)).Once()

		listener := Listener(mockMonitor)
		listener.On(mockEvent)

		mockEvent.AssertExpectations(t)
		mockMonitor.AssertExpectations(t)
	}
}

func TestListenerEventTypeNotMonitored(t *testing.T) {
	mockEvent := &mockEvent{}
	mockEvent.On("Type").Return(httppool.EventTypeStart).Once()

	mockMonitor := &mockMonitor{}

	listener := Listener(mockMonitor)
	listener.On(mockEvent)

	mockEvent.AssertExpectations(t)
	mockMonitor.AssertExpectations(t)
}
