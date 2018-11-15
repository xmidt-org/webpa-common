package xerrors

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type subError struct {
	err error
	str string
}

func (err subError) Error() string {
	return fmt.Sprintf("%s(%s)", err.str, err.err.Error())
}

type subErrorPtr struct {
	err *error
	str string
}

func (err subErrorPtr) Error() string {
	return fmt.Sprintf("%s(%s)", err.str, (*err.err).Error())
}

func TestGetErrorInt(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(nil, getError(5))

}
func TestGetErrorString(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(nil, getError("hi"))
}
func TestGetErrorNil(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(nil, getError(nil))
}

func TestGetErrorError(t *testing.T) {
	assert := assert.New(t)

	expected := errors.New("hi")
	assert.Equal(expected, getError(expected))
}
func TestGetErrorErrorPTR(t *testing.T) {
	assert := assert.New(t)

	expected := errors.New("hi")
	assert.Equal(expected, getError(&expected))
}

func TestGetErrorErrorComplex(t *testing.T) {
	assert := assert.New(t)
	type testA struct {
		error
		str string
	}
	expected := testA{errors.New("hi"), "bye"}
	assert.Equal(expected, getError(expected))
}

func TestGetErrorErrorComplexWithPointer(t *testing.T) {
	assert := assert.New(t)
	type testA struct {
		error
		str string
	}
	expected := testA{errors.New("hi"), "bye"}
	assert.Equal(expected, getError(&expected))
}

func TestGetErrorNoError(t *testing.T) {
	assert := assert.New(t)
	type testA struct {
		str string
	}
	expected := testA{"bye"}
	assert.Nil(getError(&expected))
}

func TestFirstCauseCustomSubError(t *testing.T) {
	assert := assert.New(t)
	type testA struct {
		error
		str string
	}
	exectedErr := errors.New("testA")
	test := testA{exectedErr, "cool"}
	assert.Equal(exectedErr, FirstCause(subError{test, "neat"}))
}

func TestFirstCauseNil(t *testing.T) {
	assert := assert.New(t)

	assert.Nil(FirstCause(nil))
}

func TestFirstCauseChainSubError(t *testing.T) {
	assert := assert.New(t)

	exectedErr := errors.New("expected error")
	test := subError{
		subError{
			subError{
				subErrorPtr{&exectedErr, "cool"},
				"c",
			},
			"b",
		},
		"a",
	}
	assert.Equal(exectedErr, FirstCause(subError{test, "root"}))
}

func TestGetErrorSubError(t *testing.T) {
	assert := assert.New(t)

	expected := subError{errors.New("hi"), "bye"}
	assert.Equal(expected, getError(&expected))
}

func TestFirstCauseBasic(t *testing.T) {
	assert := assert.New(t)

	err := errors.New("my bad")
	assert.Equal(err, FirstCause(err))
}

func testFirstCauseHTTPHandler(t *testing.T, serverSleep time.Duration, contextDeadline time.Duration, timeout time.Duration, useDefer bool) {
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

	xerr := FirstCause(clientErr)
	assert.Equal(clientErr.(*url.Error).Err, xerr)
}

func TestFirstCauseHTTP(t *testing.T) {

	testData := []struct {
		name            string
		serverSleep     time.Duration
		contextDeadline time.Duration
		timeout         time.Duration
		useDefer        bool
	}{
		{"client-timeout", time.Second, 500 * time.Millisecond, time.Millisecond, true},
		{"context-cancel", time.Nanosecond, 500 * time.Millisecond, time.Millisecond, false},
		{"context-timeout", time.Second, time.Millisecond, 500 * time.Millisecond, true},
	}

	for _, record := range testData {
		t.Run(fmt.Sprintf("handle/%s", record.name), func(t *testing.T) {
			testFirstCauseHTTPHandler(t, record.serverSleep, record.contextDeadline, record.timeout, record.useDefer)
		})
	}
}
