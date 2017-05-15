package device

import (
	"sync/atomic"
	"time"
)

// Statistics represents the set of tracked attributes for a single device
type Statistics struct {
	BytesReceived    uint32    `json:"bytesReceived"`
	BytesSent        uint32    `json:"bytesSent"`
	MessagesReceived uint32    `json:"messagesReceived"`
	MessagesSent     uint32    `json:"messagesSent"`
	ConnectedAt      time.Time `json:"connectedAt"`
}

func (s *Statistics) AddBytesReceived(delta uint32) {
	atomic.AddUint32(&s.BytesReceived, delta)
}

func (s *Statistics) AddBytesSent(delta uint32) {
	atomic.AddUint32(&s.BytesSent, delta)
}

func (s *Statistics) AddMessageReceived(delta uint32) {
	atomic.AddUint32(&s.MessagesReceived, delta)
}

func (s *Statistics) AddMessageSent(delta uint32) {
	atomic.AddUint32(&s.MessagesSent, delta)
}

func (s *Statistics) Copy(output *Statistics) *Statistics {
	if output == nil {
		output = new(Statistics)
	}

	output.BytesReceived = atomic.LoadUint32(&s.BytesReceived)
	output.BytesSent = atomic.LoadUint32(&s.BytesSent)
	output.MessagesReceived = atomic.LoadUint32(&s.MessagesReceived)
	output.MessagesSent = atomic.LoadUint32(&s.MessagesSent)
	output.ConnectedAt = s.ConnectedAt

	return output
}
