package wrpendpoint

import "time"

// Timing records information about the times that various components took
type Timing map[string]time.Duration

// Set sets or replaces the timing for a given component
func (t Timing) Set(component string, duration time.Duration) {
	t[component] = duration
}

// Add adds time to a component, or sets the time if that component has no timing yet
func (t Timing) Add(component string, duration time.Duration) {
	t[component] += duration
}

// Timed describes the behavior of something that is timed, possibly involving
// multiple components.
type Timed interface {
	Timing() Timing
}
