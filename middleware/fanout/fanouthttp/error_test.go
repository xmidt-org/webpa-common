package fanouthttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/stretchr/testify/assert"
)

func TestServerErrorEncoder(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			err                error
			expectedStatusCode int
			expectedHeader     http.Header
		}{
			{nil, http.StatusInternalServerError, http.Header{}},
			{errors.New("random error"), http.StatusInternalServerError, http.Header{}},
			{context.DeadlineExceeded, http.StatusGatewayTimeout, http.Header{}},
			{&httperror.E{Code: 403, Header: http.Header{"Foo": []string{"Bar"}}}, 403, http.Header{"Foo": []string{"Bar"}}},
			{tracing.NewSpanError(nil), http.StatusInternalServerError, http.Header{}},
			{tracing.NewSpanError(errors.New("random error")), http.StatusInternalServerError, http.Header{}},
			{tracing.NewSpanError(context.DeadlineExceeded), http.StatusGatewayTimeout, http.Header{}},
			{tracing.NewSpanError(&httperror.E{Code: 512, Header: http.Header{"Foo": []string{"Bar"}}}), http.StatusServiceUnavailable, http.Header{"Foo": []string{"Bar"}}},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		response := httptest.NewRecorder()
		ServerErrorEncoder("")(context.Background(), record.err, response)
		assert.Equal(record.expectedStatusCode, response.Code)
		assert.Equal(record.expectedHeader, response.Header())
	}
}

func TestHeadersForError(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			err            error
			expectedHeader http.Header
		}{
			{nil, http.Header{}},
			{errors.New("random error"), http.Header{}},
			{context.DeadlineExceeded, http.Header{}},
			{&httperror.E{Header: http.Header{"Foo": []string{"Bar"}}}, http.Header{"Foo": []string{"Bar"}}},
			{tracing.NewSpanError(nil), http.Header{}},
			{tracing.NewSpanError(errors.New("random error")), http.Header{}},
			{tracing.NewSpanError(context.DeadlineExceeded), http.Header{}},
			{tracing.NewSpanError(&httperror.E{Header: http.Header{"Foo": []string{"Bar"}}}), http.Header{"Foo": []string{"Bar"}}},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		actualHeader := make(http.Header)
		HeadersForError(record.err, "", actualHeader)
		assert.Equal(record.expectedHeader, actualHeader)
	}
}

func TestStatusCodeForError(t *testing.T) {
	var (
		assert  = assert.New(t)
		spanner = tracing.NewSpanner()

		testData = []struct {
			err                error
			expectedStatusCode int
		}{
			{nil, http.StatusInternalServerError},
			{errors.New("random error"), http.StatusInternalServerError},
			{context.DeadlineExceeded, http.StatusGatewayTimeout},
			{&httperror.E{Code: 403}, 403},
			{tracing.NewSpanError(nil), http.StatusInternalServerError},
			{tracing.NewSpanError(errors.New("random error")), http.StatusInternalServerError},
			{tracing.NewSpanError(context.DeadlineExceeded), http.StatusGatewayTimeout},
			{tracing.NewSpanError(&httperror.E{Code: 403}), 403},
			{tracing.NewSpanError(nil), http.StatusInternalServerError},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(context.DeadlineExceeded),
				),
				http.StatusServiceUnavailable,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(context.DeadlineExceeded),
					spanner.Start("2")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("3")(&httperror.E{Code: http.StatusInternalServerError}),
				),
				http.StatusServiceUnavailable,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(&httperror.E{Code: http.StatusGatewayTimeout}),
				),
				http.StatusServiceUnavailable,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("2")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("3")(&httperror.E{Code: http.StatusGatewayTimeout}),
				),
				http.StatusServiceUnavailable,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(context.DeadlineExceeded),
					spanner.Start("2")(&httperror.E{Code: http.StatusNotFound}),
				),
				http.StatusNotFound,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(&httperror.E{Code: http.StatusNotFound}),
					spanner.Start("2")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("3")(&httperror.E{Code: http.StatusInternalServerError}),
				),
				http.StatusNotFound,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("2")(&httperror.E{Code: http.StatusNotFound}),
					spanner.Start("3")(&httperror.E{Code: http.StatusInternalServerError}),
				),
				http.StatusNotFound,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(&httperror.E{Code: http.StatusInternalServerError}),
					spanner.Start("2")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("3")(&httperror.E{Code: http.StatusNotFound}),
				),
				http.StatusNotFound,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		assert.Equal(record.expectedStatusCode, StatusCodeForError(record.err))
	}
}
