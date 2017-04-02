package device

import (
	"errors"
	"fmt"
)

var (
	ErrorDeviceNotFound               = errors.New("The device does not exist")
	ErrorNonUniqueID                  = errors.New("More than once device with that identifier is connected")
	ErrorInvalidTransactionKey        = errors.New("Transaction keys must be non-empty strings")
	ErrorNoSuchTransactionKey         = errors.New("That transaction key is not registered")
	ErrorTransactionAlreadyRegistered = errors.New("That transaction is already registered")
	ErrorTransactionCancelled         = errors.New("The transaction has been cancelled")
	ErrorResponseNoContents           = errors.New("The response has no contents")
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
