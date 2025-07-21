// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
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
	BytesReceived() int

	// AddBytesReceived increments the BytesReceived count
	AddBytesReceived(int)

	// MessagesReceived returns the total messages received since this instance was created
	MessagesReceived() int

	// AddMessagesReceived increments the MessagesReceived count
	AddMessagesReceived(int)

	// BytesSent returns the total bytes sent since this instance was created
	BytesSent() int

	// AddBytesSent increments the BytesSent count
	AddBytesSent(int)

	// MessagesSent returns the total messages sent since this instance was created
	MessagesSent() int

	// AddMessagesSent increments the MessagesSent count
	AddMessagesSent(int)

	// Duplications returns the number of times this device has had a duplicate connected, i.e.
	// a device with the same device ID.
	Duplications() int

	// AddDuplications increments the count of duplications
	AddDuplications(int)

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
		formattedConnectedAt: connectedAt.Format(time.RFC3339Nano),
	}
}

// statistics is the internal Statistics implementation
type statistics struct {
	lock sync.RWMutex

	bytesReceived    int
	bytesSent        int
	messagesReceived int
	messagesSent     int
	duplications     int

	now                  func() time.Time
	connectedAt          time.Time
	formattedConnectedAt string
}

func (s *statistics) BytesReceived() int {
	s.lock.RLock()
	var result = s.bytesReceived
	s.lock.RUnlock()

	return result
}

func (s *statistics) AddBytesReceived(delta int) {
	s.lock.Lock()
	s.bytesReceived += delta
	s.lock.Unlock()
}

func (s *statistics) BytesSent() int {
	s.lock.RLock()
	var result = s.bytesSent
	s.lock.RUnlock()

	return result
}

func (s *statistics) AddBytesSent(delta int) {
	s.lock.Lock()
	s.bytesSent += delta
	s.lock.Unlock()
}

func (s *statistics) MessagesReceived() int {
	s.lock.RLock()
	var result = s.messagesReceived
	s.lock.RUnlock()

	return result
}

func (s *statistics) AddMessagesReceived(delta int) {
	s.lock.Lock()
	s.messagesReceived += delta
	s.lock.Unlock()
}

func (s *statistics) MessagesSent() int {
	s.lock.RLock()
	var result = s.messagesSent
	s.lock.RUnlock()

	return result
}

func (s *statistics) AddMessagesSent(delta int) {
	s.lock.Lock()
	s.messagesSent += delta
	s.lock.Unlock()
}

func (s *statistics) Duplications() int {
	s.lock.RLock()
	var result = s.duplications
	s.lock.RUnlock()

	return result
}

func (s *statistics) AddDuplications(delta int) {
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
	if data, err := s.MarshalJSON(); err == nil {
		return string(data)
	} else {
		return err.Error()
	}
}

func (s *statistics) MarshalJSON() ([]byte, error) {
	s.lock.RLock()
	output := []byte(fmt.Sprintf(
		`{"bytesSent": %d, "messagesSent": %d, "bytesReceived": %d, "messagesReceived": %d, "duplications": %d, "connectedAt": "%s", "upTime": "%s"}`,
		s.bytesSent,
		s.messagesSent,
		s.bytesReceived,
		s.messagesReceived,
		s.duplications,
		s.formattedConnectedAt,
		s.UpTime(),
	))
	s.lock.RUnlock()
	return output, nil
}
