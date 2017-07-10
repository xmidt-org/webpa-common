package device

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"time"
)

// Statistics represents a set of device statistics.
type Statistics interface {
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

	// ConnectedAt returns the connection time at which this statistics began tracking
	ConnectedAt() time.Time
}

// NewStatistics creates a Statistics instance with the given connection time
func NewStatistics(connectedAt time.Time) Statistics {
	return &statistics{
		connectedAt: connectedAt,
	}
}

// statistics is the internal Statistics implementation
type statistics struct {
	bytesReceived    uint32
	bytesSent        uint32
	messagesReceived uint32
	messagesSent     uint32
	connectedAt      time.Time
}

func (s *statistics) BytesReceived() uint32 {
	return atomic.LoadUint32(&s.bytesReceived)
}

func (s *statistics) AddBytesReceived(delta uint32) {
	atomic.AddUint32(&s.bytesReceived, delta)
}

func (s *statistics) MessagesReceived() uint32 {
	return atomic.LoadUint32(&s.messagesReceived)
}

func (s *statistics) AddMessagesReceived(delta uint32) {
	atomic.AddUint32(&s.messagesReceived, delta)
}

func (s *statistics) BytesSent() uint32 {
	return atomic.LoadUint32(&s.bytesSent)
}

func (s *statistics) AddBytesSent(delta uint32) {
	atomic.AddUint32(&s.bytesSent, delta)
}

func (s *statistics) MessagesSent() uint32 {
	return atomic.LoadUint32(&s.messagesSent)
}

func (s *statistics) AddMessagesSent(delta uint32) {
	atomic.AddUint32(&s.messagesSent, delta)
}

func (s *statistics) ConnectedAt() time.Time {
	return s.connectedAt
}

func (s *statistics) String() string {
	data, _ := s.MarshalJSON()
	return string(data)
}

func (s *statistics) MarshalJSON() ([]byte, error) {
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
