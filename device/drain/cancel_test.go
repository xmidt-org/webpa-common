package drain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testCancelNotActive(t *testing.T) {
	var (
		assert = assert.New(t)

		d      = new(mockDrainer)
		cancel = Cancel{d}

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	// nolint: typecheck
	d.On("Cancel").Return((<-chan struct{})(nil), ErrNotActive).Once()
	cancel.ServeHTTP(response, request)
	assert.Equal(http.StatusConflict, response.Code)

	// nolint: typecheck
	d.AssertExpectations(t)
}

func testCancelSuccess(t *testing.T) {
	var (
		assert = assert.New(t)

		d          = new(mockDrainer)
		cancel     = Cancel{d}
		done       = make(chan struct{})
		cancelWait = make(chan time.Time)
		serveHTTP  = make(chan struct{})

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	// nolint: typecheck
	d.On("Cancel").WaitUntil(cancelWait).Return((<-chan struct{})(done), error(nil)).Once()

	go func() {
		defer close(serveHTTP)
		cancel.ServeHTTP(response, request)
	}()

	cancelWait <- time.Time{}
	close(done)
	select {
	case <-serveHTTP:
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("ServeHTTP did not return")
		return
	}

	assert.Equal(http.StatusOK, response.Code)
	// nolint: typecheck
	d.AssertExpectations(t)
}

func testCancelTimeout(t *testing.T) {
	var (
		assert = assert.New(t)

		d          = new(mockDrainer)
		cancel     = Cancel{d}
		done       = make(chan struct{})
		cancelWait = make(chan time.Time)
		serveHTTP  = make(chan struct{})

		ctx, ctxCancel = context.WithCancel(context.Background())
		response       = httptest.NewRecorder()
		request        = httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	)

	// nolint: typecheck
	d.On("Cancel").WaitUntil(cancelWait).Return((<-chan struct{})(done), error(nil)).Once()

	go func() {
		defer close(serveHTTP)
		cancel.ServeHTTP(response, request)
	}()

	cancelWait <- time.Time{}
	ctxCancel()
	select {
	case <-serveHTTP:
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("ServeHTTP did not return")
		return
	}

	assert.Equal(http.StatusOK, response.Code)
	// nolint: typecheck
	d.AssertExpectations(t)
}

func TestCancel(t *testing.T) {
	t.Run("NotActive", testCancelNotActive)
	t.Run("Success", testCancelSuccess)
	t.Run("Timeout", testCancelTimeout)
}
