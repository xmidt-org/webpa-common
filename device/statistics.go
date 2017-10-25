package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Statistics represents a set of device statistics.
type Statistics interface {
	fmt.Stringer
	json.Marshaler

	// BytesReceived returns the total bytes received since this instance was created
	BytesReceived() uint32

	// AddBytesReceived adds a certain number of bytes to the BytesReceived count.
	// Implementations will always be safe for concurrent access.
	AddBytesReceived(uint32)

	// MessagesReceived returns the total messages received since this instance was created
	MessagesReceived() uint32

	// AddMessagesReceived adds a certain number of messages to the MessagesReceived count.
	// Implementations will always be safe for concurrent access.
	AddMessagesReceived(uint32)

	// BytesSent returns the total bytes sent since this instance was created
	BytesSent() uint32

	// AddBytesSent adds a certain number of bytes to the BytesSent count.
	// Implementations will always be safe for concurrent access.
	AddBytesSent(uint32)

	// MessagesSent returns the total messages sent since this instance was created
	MessagesSent() uint32

	// AddMessagesSent adds a certain number of messages to the MessagesSent count.
	// Implementations will always be safe for concurrent access.
	AddMessagesSent(uint32)

	// Duplications returns the number of times this device has had a duplicate connected, i.e.
	// a device with the same device ID.
	Duplications() uint32

	// AddDuplications increments the count of duplications
	AddDuplications(uint32)

	// ConnectedAt returns the connection time at which this statistics began tracking
	ConnectedAt() time.Time

	// UpTime computes the duration for which the device has been connected
	UpTime() time.Duration
}

// NewStatistics creates a Statistics instance with the given connection time
// If now is nil, this method uses time.Now.
func NewStatistics(now func() time.Time, connectedAt time.Time) Statistics {
	if now == nil {
		now = time.Now
	}

	connectedAt = connectedAt.UTC()
	return &statistics{
		now:                  now,
		connectedAt:          connectedAt,
		formattedConnectedAt: connectedAt.Format(time.RFC3339),
	}
}

// statistics is the internal Statistics implementation
type statistics struct {
	lock sync.RWMutex

	bytesReceived    uint32
	bytesSent        uint32
	messagesReceived uint32
	messagesSent     uint32
	duplications     uint32

	now                  func() time.Time
	connectedAt          time.Time
	formattedConnectedAt string
}

func (s *statistics) BytesReceived() uint32 {
	s.lock.RLock()
	var result = s.bytesReceived
	s.lock.RUnlock()

	return result
}

func (s *statistics) AddBytesReceived(delta uint32) {
	s.lock.Lock()
	s.bytesReceived += delta
	s.lock.Unlock()
}

func (s *statistics) BytesSent() uint32 {
	s.lock.RLock()
	var result = s.bytesSent
	s.lock.RUnlock()

	return result
}

func (s *statistics) AddBytesSent(delta uint32) {
	s.lock.Lock()
	s.bytesSent += delta
	s.lock.Unlock()
}

func (s *statistics) MessagesReceived() uint32 {
	s.lock.RLock()
	var result = s.messagesReceived
	s.lock.RUnlock()

	return result
}

func (s *statistics) AddMessagesReceived(delta uint32) {
	s.lock.Lock()
	s.messagesReceived += delta
	s.lock.Unlock()
}

func (s *statistics) MessagesSent() uint32 {
	s.lock.RLock()
	var result = s.messagesSent
	s.lock.RUnlock()

	return result
}

func (s *statistics) AddMessagesSent(delta uint32) {
	s.lock.Lock()
	s.messagesSent += delta
	s.lock.Unlock()
}

func (s *statistics) Duplications() uint32 {
	s.lock.RLock()
	var result = s.duplications
	s.lock.RUnlock()

	return result
}

func (s *statistics) AddDuplications(delta uint32) {
	s.lock.Lock()
	s.duplications += delta
	s.lock.Unlock()
}

func (s *statistics) ConnectedAt() time.Time {
	return s.connectedAt
}

func (s *statistics) UpTime() time.Duration {
	return s.now().Sub(s.connectedAt)
}

func (s *statistics) String() string {
	data, _ := s.MarshalJSON()
	return string(data)
}

func (s *statistics) MarshalJSON() ([]byte, error) {
	output := bytes.NewBuffer(make([]byte, 0, 150))
	s.lock.RLock()
	fmt.Fprintf(
		output,
		`{"bytesSent": %d, "messagesSent": %d, "bytesReceived": %d, "messagesReceived": %d, "duplications": %d, "connectedAt": "%s", "upTime": "%s"}`,
		s.bytesSent,
		s.messagesSent,
		s.bytesReceived,
		s.messagesReceived,
		s.duplications,
		s.formattedConnectedAt,
		s.UpTime(),
	)

	s.lock.RUnlock()
	return output.Bytes(), nil
}
