package drain

import (
	"sync/atomic"
	"time"

	"github.com/Comcast/webpa-common/xmetrics"
)

// Progress represents a snapshot of what a drain job has done so far.
type Progress struct {
	// Visited is the number of devices handled so far.  This value will not
	// exceed the Job.Count value.
	Visited int `json:"visited"`

	// Drained is the count of visited devices that have actually been disconnected
	// due to the drain.  Devices can disconnect or be disconnected outside a drain job,
	// so this value can be lower than Visited, even in a job that has finished.
	Drained int `json:"drained"`

	// Started is the UTC system time at which the drain job was started.
	Started time.Time `json:"started"`

	// Finished is the UTC system time at which the drain job finished or was canceled.
	// If the job is running, this field will be nil.
	Finished *time.Time `json:"finished,omitempty"`
}

type tracker struct {
	visited  int32
	drained  int32
	started  time.Time
	finished int64
	counter  xmetrics.Adder
}

func (t *tracker) Progress() Progress {
	var (
		finished = atomic.LoadInt64(&t.finished)
		p        = Progress{
			Visited: int(atomic.LoadInt32(&t.visited)),
			Drained: int(atomic.LoadInt32(&t.drained)),
			Started: t.started,
		}
	)

	if finished > 0 {
		timestamp := time.Unix(finished, 0)
		p.Finished = &timestamp
	}

	return p
}

func (t *tracker) addVisited(delta int) {
	atomic.AddInt32(&t.visited, int32(delta))
}

func (t *tracker) addDrained(delta int) {
	atomic.AddInt32(&t.drained, int32(delta))
	t.counter.Add(float64(delta))
}

func (t *tracker) done(timestamp time.Time) {
	atomic.StoreInt64(&t.finished, timestamp.UTC().Unix())
}
