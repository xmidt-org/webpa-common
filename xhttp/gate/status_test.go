// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/sallust"
)

func testStatusServeHTTP(t *testing.T, state bool) {
	var (
		assert            = assert.New(t)
		logger            = sallust.Default()
		ctx               = sallust.With(context.Background(), logger)
		expectedTimestamp = time.Now()
		expectedStatus    = fmt.Sprintf(`{"open": %t, "timestamp": "%s"}`, state, expectedTimestamp.UTC().Format(time.RFC3339))

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		g      = New(state)
		status = Status{Gate: g}
	)

	g.(*gate).now = func() time.Time { return expectedTimestamp }

	status.ServeHTTP(response, request.WithContext(ctx))
	assert.Equal(http.StatusOK, response.Code)
	assert.JSONEq(
		expectedStatus,
		response.Body.String(),
	)
}

func TestStatus(t *testing.T) {
	t.Run("Open", func(t *testing.T) {
		testStatusServeHTTP(t, true)
	})

	t.Run("Closed", func(t *testing.T) {
		testStatusServeHTTP(t, false)
	})
}
