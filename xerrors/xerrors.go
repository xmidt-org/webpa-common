package xerrors

import (
	"net"
	"net/url"
	"strings"
)

//go:generate stringer -type=ErrorType

type ErrorType int64

const (
	UnknownError ErrorType = iota
	URLError
	NetError
	RequestCanceledError
	RequestCanceledConError
	TemporaryError
	TimeoutError
	ContextDeadlineExceededError
	ContextCanceledError
)

type SearchString struct {
	str       string
	errBucket ErrorType
}

var (
	StringSearchs = []SearchString{
		{"request canceled", RequestCanceledError},
		{"request canceled while waiting for connection", RequestCanceledConError},
		{"context deadline exceeded", ContextDeadlineExceededError},
		{"context canceled", ContextCanceledError},
	}
)

type ErrorBucket map[ErrorType]struct{}

func (set ErrorBucket) String() string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k.String())
	}
	return "(" + strings.Join(keys, ", ") + ")"
}

type XError struct {
	ErrorBucket ErrorBucket
	Error       error
}

func GetXError(err error) *XError {
	xerror := XError{
		ErrorBucket: make(ErrorBucket),
		Error:       err,
	}

	for _, item := range StringSearchs {
		if strings.Contains(err.Error(), item.str) {
			xerror.ErrorBucket[item.errBucket] = struct{}{}
		}
	}

	switch err := err.(type) {
	case *url.Error:
		xerror.ErrorBucket[URLError] = struct{}{}
		if err.Timeout() {
			xerror.ErrorBucket[TimeoutError] = struct{}{}
		}
		if err.Temporary() {
			xerror.ErrorBucket[TemporaryError] = struct{}{}
		}
	case net.Error:
		xerror.ErrorBucket[NetError] = struct{}{}
		if err.Timeout() {
			xerror.ErrorBucket[TimeoutError] = struct{}{}
		}
		if err.Temporary() {
			xerror.ErrorBucket[TemporaryError] = struct{}{}
		}
	}

	if len(xerror.ErrorBucket) == 0 {
		xerror.ErrorBucket[UnknownError] = struct{}{}
	}
	return &xerror
}

func (xerr *XError) IsContextCanceled() bool {
	_, ok := xerr.ErrorBucket[ContextCanceledError]
	return ok
}

func (xerr *XError) IsContextTimeout() bool {
	_, ok := xerr.ErrorBucket[ContextDeadlineExceededError]
	return ok
}

func (xerr *XError) IsClientTimeout() bool {
	_, reqCancled := xerr.ErrorBucket[RequestCanceledError]
	_, timeout := xerr.ErrorBucket[TimeoutError]
	return reqCancled && timeout
}
