package device

import (
	"errors"
)

var (
	ErrorInvalidDeviceName            = errors.New("Invalid device name")
	ErrorDeviceNotFound               = errors.New("The device does not exist")
	ErrorNonUniqueID                  = errors.New("More than once device with that identifier is connected")
	ErrorDuplicateKey                 = errors.New("That key is a duplicate")
	ErrorInvalidTransactionKey        = errors.New("Transaction keys must be non-empty strings")
	ErrorNoSuchTransactionKey         = errors.New("That transaction key is not registered")
	ErrorTransactionAlreadyRegistered = errors.New("That transaction is already registered")
	ErrorTransactionCancelled         = errors.New("The transaction has been cancelled")
	ErrorResponseNoContents           = errors.New("The response has no contents")
	ErrorDeviceBusy                   = errors.New("That device is busy")
	ErrorDeviceClosed                 = errors.New("That device has been closed")
)
