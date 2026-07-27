// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"errors"
)

var (
	ErrorMissingDeviceNameContext     = errors.New("missing device ID in request context")
	ErrorMissingSecureContext         = errors.New("missing security information in request context")
	ErrorMissingDeviceNameHeader      = errors.New("missing device name header")
	ErrorMissingDeviceNameVar         = errors.New("missing device name path variable")
	ErrorMissingPathVars              = errors.New("missing URI path variables")
	ErrorInvalidDeviceName            = errors.New("invalid device name")
	ErrorDeviceNotFound               = errors.New("the device does not exist")
	ErrorNonUniqueID                  = errors.New("more than once device with that identifier is connected")
	ErrorDuplicateKey                 = errors.New("that key is a duplicate")
	ErrorDuplicateDevice              = errors.New("that device is already in this registry")
	ErrorInvalidTransactionKey        = errors.New("transaction keys must be non-empty strings")
	ErrorNoSuchTransactionKey         = errors.New("that transaction key is not registered")
	ErrorTransactionAlreadyRegistered = errors.New("that transaction is already registered")
	ErrorTransactionCanceled          = errors.New("the transaction has been canceled")
	ErrorResponseNoContents           = errors.New("the response has no contents")
	ErrorDeviceBusy                   = errors.New("that device is busy")
	ErrorDeviceClosed                 = errors.New("that device has been closed")
	ErrorTransactionsClosed           = errors.New("transactions are closed for that device")
	ErrorTransactionsAlreadyClosed    = errors.New("that Transactions is already closed")
	ErrorDeviceFilteredOut            = errors.New("device blocked from connecting due to filters")
)
