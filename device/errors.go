package device

import (
	"fmt"
)

// DeviceError is the common interface implemented by all error objects
// which carry device-related metadata
type DeviceError interface {
	error
	ID() ID
	Key() Key
}

// deviceError is the internal DeviceError implementation
type deviceError struct {
	id   ID
	key  Key
	text string
}

func (e *deviceError) ID() ID {
	return e.id
}

func (e *deviceError) Key() Key {
	return e.key
}

func (e *deviceError) Error() string {
	return e.text
}

func newDeviceError(id ID, key Key, message string) DeviceError {
	return &deviceError{
		id:   id,
		key:  key,
		text: fmt.Sprintf("Device [id=%s, key=%s]: %s", id, key, message),
	}
}

func NewClosedError(id ID, key Key) DeviceError {
	return newDeviceError(id, key, "closed")
}

func NewBusyError(id ID, key Key) DeviceError {
	return newDeviceError(id, key, "busy")
}

func NewDuplicateKeyError(key Key) DeviceError {
	return newDeviceError("", key, "duplicate key")
}

func NewDeviceNotFoundError(id ID) DeviceError {
	return newDeviceError(id, "", "not found")
}
