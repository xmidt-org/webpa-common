package monitor

var (
	eventCountKey interface{} = "eventCount"
)

// EventCountKey returns the contextual logging key for the event count
func EventCountKey() interface{} {
	return eventCountKey
}
