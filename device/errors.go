package device

import (
	"fmt"
)

// deviceError exposes the basic error metadata
type deviceError struct {
	id  ID
	key Key
}

func (e *deviceError) ID() ID {
	return e.id
}

func (e *deviceError) Key() Key {
	return e.key
}

// ClosedError indicates that an operation was attempted on
// a closed device that is not allowed, e.g. sending a message
type ClosedError struct {
	deviceError
}

func (e *ClosedError) Error() string {
	return fmt.Sprintf("Device [%s] with key [%s] is closed", e.id, e.key)
}

func NewClosedError(id ID, key Key) *ClosedError {
	return &ClosedError{
		deviceError{
			id:  id,
			key: key,
		},
	}
}

// BusyError indicates that a device's message queue is full
type BusyError struct {
	deviceError
}

func (e *BusyError) Error() string {
	return fmt.Sprintf("Device [%s] with key [%s] is busy", e.id, e.key)
}

func NewBusyError(id ID, key Key) *BusyError {
	return &BusyError{
		deviceError{
			id:  id,
			key: key,
		},
	}
}

type MissingIDError struct {
	deviceError
}

func (e *MissingIDError) Error() string {
	return fmt.Sprintf("No device exists with id [%s]", e.id)
}

func NewMissingIDError(id ID) *MissingIDError {
	return &MissingIDError{
		deviceError{
			id: id,
		},
	}
}

type MissingKeyError struct {
	deviceError
}

func (e *MissingKeyError) Error() string {
	return fmt.Sprintf("No device exists with key [%s]", e.key)
}

func NewMissingKeyError(key Key) *MissingKeyError {
	return &MissingKeyError{
		deviceError{
			key: key,
		},
	}
}

type DuplicateKeyError struct {
	deviceError
}

func (e *DuplicateKeyError) Error() string {
	return fmt.Sprintf("Duplicate key [%s]", e.key)
}

func NewDuplicateKeyError(key Key) *DuplicateKeyError {
	return &DuplicateKeyError{
		deviceError{
			key: key,
		},
	}
}
