// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import "net/http"

// Client is an interface implemented by net/http.Client
type Client interface {
	Do(*http.Request) (*http.Response, error)
}

var _ Client = (*http.Client)(nil)
