// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"net/http"
	"time"
)

// Status is an http.Handler that reports the status of a gate
type Status struct {
	Gate Interface
}

func (s *Status) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	state, timestamp := s.Gate.State()
	fmt.Fprintf(response, `{"open": %t, "timestamp": "%s"}`, state, timestamp.Format(time.RFC3339))
}
