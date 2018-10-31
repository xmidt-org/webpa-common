package clock

import "time"

// Interface represents a clock with the same core functionality available as in the stdlib time package
type Interface interface {
	Now() time.Time
	Sleep(time.Duration)
	NewTicker(time.Duration) Ticker
	NewTimer(time.Duration) Timer
}

type systemClock struct{}

func (sc systemClock) Now() time.Time {
	return time.Now()
}

func (sc systemClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (sc systemClock) NewTicker(d time.Duration) Ticker {
	return systemTicker{time.NewTicker(d)}
}

func (sc systemClock) NewTimer(d time.Duration) Timer {
	return systemTimer{time.NewTimer(d)}
}

// System returns a clock backed by the time package
func System() Interface {
	return systemClock{}
}
