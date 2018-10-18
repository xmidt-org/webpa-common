package monitor

import (
	"testing"

	"github.com/Comcast/webpa-common/capacitor/capacitortest"
	"github.com/stretchr/testify/mock"
)

func TestDelayedListener(t *testing.T) {
	var (
		expectedEvent = Event{Key: "this is a test key"}

		c  = new(capacitortest.Mock)
		l  = new(mockListener)
		dl = DelayedListener{l, c}
	)

	l.On("MonitorEvent", expectedEvent).Once()
	c.On("Submit", mock.MatchedBy(func(func()) bool { return true })).Once().Run(func(arguments mock.Arguments) {
		arguments.Get(0).(func())()
	})

	dl.MonitorEvent(expectedEvent)

	c.AssertExpectations(t)
	l.AssertExpectations(t)
}
