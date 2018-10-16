package clock

import "time"

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type systemTicker struct {
	*time.Ticker
}

func (st systemTicker) C() <-chan time.Time {
	return st.Ticker.C
}

func WrapTicker(t *time.Ticker) Ticker {
	return systemTicker{t}
}
