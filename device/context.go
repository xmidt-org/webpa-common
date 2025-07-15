// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"context"
	"net/http"
)

type key int

const (
	idKey key = iota
	metadataKey
)

// GetID returns the device ID from the context if any.
func GetID(ctx context.Context) (id ID, ok bool) {
	id, ok = ctx.Value(idKey).(ID)
	return
}

// WithID returns a new context with the given device ID as a value.
func WithID(parent context.Context, id ID) context.Context {
	return context.WithValue(parent, idKey, id)
}

// WithIDRequest returns a new HTTP request with the given device ID in the associated Context.
func WithIDRequest(id ID, original *http.Request) *http.Request {
	return original.WithContext(
		WithID(original.Context(), id),
	)
}

// WithDeviceMetadata returns a new context with the given metadata as a value.
func WithDeviceMetadata(parent context.Context, metadata *Metadata) context.Context {
	return context.WithValue(parent, metadataKey, metadata)
}

// GetDeviceMetadata returns the device metadata from the context if any.
func GetDeviceMetadata(ctx context.Context) (metadata *Metadata, ok bool) {
	metadata, ok = ctx.Value(metadataKey).(*Metadata)
	return
}
