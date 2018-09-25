package xerrors

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetXErrorBasic(t *testing.T) {
	assert := assert.New(t)

	err := errors.New("my bad")
	expected := &XError{ErrorBucket{UnknownError: {}}, err}
	assert.Equal(expected, GetXError(err))
}

func testXErrorHandler(t *testing.T, serverSleep time.Duration, contextDeadline time.Duration, timeout time.Duration, useDefer bool, expectedErrorSet ErrorBucket) *XError {
	assert := assert.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(serverSleep)
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	client := http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	assert.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), contextDeadline)
	if useDefer {
		defer cancel()
	} else {
		cancel() // cancel() is a hook to cancel the deadline
	}

	reqWithDeadline := req.WithContext(ctx)

	_, clientErr := client.Do(reqWithDeadline)

	if !assert.Error(clientErr) {
		assert.FailNow("clientErr can not be nil to continue")
	}

	xerr := GetXError(clientErr)
	assert.Equal(expectedErrorSet, xerr.ErrorBucket)

	t.Logf("%#v\n", clientErr)
	t.Log(xerr)
	return xerr
}

func TestGetXError(t *testing.T) {

	testData := []struct {
		name             string
		serverSleep      time.Duration
		contextDeadline  time.Duration
		timeout          time.Duration
		useDefer         bool
		expectedErrorSet ErrorBucket
	}{
		{"client-timeout", time.Second, 5 * time.Millisecond, time.Millisecond, true, ErrorBucket{RequestCanceledError: {}, TemporaryError: {}, TimeoutError: {}, URLError: {}}},
		{"context-cancel", time.Nanosecond, 5 * time.Millisecond, time.Millisecond, false, ErrorBucket{ContextCanceledError: {}, URLError: {}}},
		{"context-timeout", time.Second, time.Millisecond, 5 * time.Millisecond, true, ErrorBucket{ContextDeadlineExceededError: {}, TemporaryError: {}, TimeoutError: {}, URLError: {}}},
	}

	for _, record := range testData {
		t.Run(fmt.Sprintf("handle/%s", record.name), func(t *testing.T) {
			xerr := testXErrorHandler(t, record.serverSleep, record.contextDeadline, record.timeout, record.useDefer, record.expectedErrorSet)
			count := 0
			if xerr.IsClientTimeout(){
				count ++
			}
			if xerr.IsContextCanceled(){
				count ++
			}
			if xerr.IsContextTimeout(){
				count ++
			}
			assert.Equal(t, 1, count, "error can't be context timeout context canceled and client timeout")
		})
	}

}
