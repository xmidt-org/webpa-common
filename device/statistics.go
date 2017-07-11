package device

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"time"
)

// NewStatistics creates a Statistics instance with the given connection time.
// The typical use case is to pass time.Now() as the connection time.
func NewStatistics(connectedAt time.Time) *Statistics {
	return &Statistics{
		connectedAt: connectedAt,
	}
}

// Statistics represents a set of device statistics that is safe for concurrent access,
// including marshalling and unmarshalling as JSON.
type Statistics struct {
	bytesReceived    uint32
	bytesSent        uint32
	messagesReceived uint32
	messagesSent     uint32
	connectedAt      time.Time
}

// BytesReceived returns the total bytes received since this instance was created
func (s *Statistics) BytesReceived() uint32 {
	return atomic.LoadUint32(&s.bytesReceived)
}

// AddBytesReceived adds a certain number of bytes to the BytesReceived count
func (s *Statistics) AddBytesReceived(delta uint32) {
	atomic.AddUint32(&s.bytesReceived, delta)
}

// MessagesReceived returns the total messages received since this instance was created
func (s *Statistics) MessagesReceived() uint32 {
	return atomic.LoadUint32(&s.messagesReceived)
}

// AddMessagesReceived adds a certain number of messages to the MessagesReceived count
func (s *Statistics) AddMessagesReceived(delta uint32) {
	atomic.AddUint32(&s.messagesReceived, delta)
}

// BytesSent returns the total bytes sent since this instance was created
func (s *Statistics) BytesSent() uint32 {
	return atomic.LoadUint32(&s.bytesSent)
}

// AddBytesSent adds a certain number of bytes to the BytesSent count
func (s *Statistics) AddBytesSent(delta uint32) {
	atomic.AddUint32(&s.bytesSent, delta)
}

// MessagesSent returns the total messages sent since this instance was created
func (s *Statistics) MessagesSent() uint32 {
	return atomic.LoadUint32(&s.messagesSent)
}

// AddMessagesSent adds a certain number of messages to the MessagesSent count
func (s *Statistics) AddMessagesSent(delta uint32) {
	atomic.AddUint32(&s.messagesSent, delta)
}

// ConnectedAt returns the connection time at which this statistics began tracking
func (s *Statistics) ConnectedAt() time.Time {
	return s.connectedAt
}

func (s *Statistics) String() string {
	data, _ := s.MarshalJSON()
	return string(data)
}

func (s *Statistics) MarshalJSON() ([]byte, error) {
	output := bytes.NewBuffer(make([]byte, 0, 150))
	fmt.Fprintf(
		output,
		`{"bytesSent": %d, "messagesSent": %d, "bytesReceived": %d, "messagesReceived": %d, "connectedAt": "%s"}`,
		s.BytesSent(),
		s.MessagesSent(),
		s.BytesReceived(),
		s.MessagesReceived(),
		s.ConnectedAt().Format(time.RFC3339),
	)

	return output.Bytes(), nil
}
