package monitor

var (
	eventCountKey string = "eventCount"
)

// EventCountKey returns the contextual logging key for the event count
func EventCountKey() string {
	return eventCountKey
}
