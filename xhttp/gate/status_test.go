package gate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
)

func testStatusServeHTTP(t *testing.T, state uint32) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)
		ctx    = logging.WithLogger(context.Background(), logger)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		gate   = New(state)
		status = Status{Gate: gate}
	)

	status.ServeHTTP(response, request.WithContext(ctx))
	assert.Equal(http.StatusOK, response.Code)
	assert.JSONEq(
		fmt.Sprintf(`{"open": %t}`, gate.Open()),
		response.Body.String(),
	)
}

func TestStatus(t *testing.T) {
	t.Run("Open", func(t *testing.T) {
		testStatusServeHTTP(t, Open)
	})

	t.Run("Closed", func(t *testing.T) {
		testStatusServeHTTP(t, Closed)
	})
}
