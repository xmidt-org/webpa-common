package gate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
)

func testLeverServeHTTPBadForm(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)
		ctx    = logging.WithLogger(context.Background(), logger)

		response = httptest.NewRecorder()
		request  = &http.Request{
			URL: &url.URL{
				RawQuery: `this!is%bad&%TT`,
			},
		}

		lever = Lever{}
	)

	lever.ServeHTTP(response, request.WithContext(ctx))
	assert.Equal(http.StatusBadRequest, response.Code)
}

func testLeverServeHTTPNoParameter(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)
		ctx    = logging.WithLogger(context.Background(), logger)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/", nil)

		lever = Lever{Parameter: "open"}
	)

	lever.ServeHTTP(response, request.WithContext(ctx))
	assert.Equal(http.StatusBadRequest, response.Code)
}

func testLeverServeHTTPBadParameter(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)
		ctx    = logging.WithLogger(context.Background(), logger)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo?open=thisisnotabool", nil)

		lever = Lever{Parameter: "open"}
	)

	lever.ServeHTTP(response, request.WithContext(ctx))
	assert.Equal(http.StatusBadRequest, response.Code)
}

func testLeverServeHTTPRaise(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)
		ctx    = logging.WithLogger(context.Background(), logger)

		gate  = New(Open)
		lever = Lever{Gate: gate, Parameter: "open"}
	)

	{
		var (
			response = httptest.NewRecorder()
			request  = httptest.NewRequest("POST", "/foo?open=true", nil)
		)

		lever.ServeHTTP(response, request.WithContext(ctx))
		assert.Equal(http.StatusOK, response.Code)
		assert.True(gate.Open())
	}

	{
		var (
			response = httptest.NewRecorder()
			request  = httptest.NewRequest("POST", "/foo?open=false", nil)
		)

		lever.ServeHTTP(response, request.WithContext(ctx))
		assert.Equal(http.StatusCreated, response.Code)
		assert.False(gate.Open())
	}

	{
		var (
			response = httptest.NewRecorder()
			request  = httptest.NewRequest("POST", "/foo?open=true", nil)
		)

		lever.ServeHTTP(response, request.WithContext(ctx))
		assert.Equal(http.StatusCreated, response.Code)
		assert.True(gate.Open())
	}

	{
		var (
			response = httptest.NewRecorder()
			request  = httptest.NewRequest("POST", "/foo?open=true", nil)
		)

		lever.ServeHTTP(response, request.WithContext(ctx))
		assert.Equal(http.StatusOK, response.Code)
		assert.True(gate.Open())
	}
}

func testLeverServeHTTPLower(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)
		ctx    = logging.WithLogger(context.Background(), logger)

		gate  = New(Closed)
		lever = Lever{Gate: gate, Parameter: "open"}
	)

	{
		var (
			response = httptest.NewRecorder()
			request  = httptest.NewRequest("POST", "/foo?open=false", nil)
		)

		lever.ServeHTTP(response, request.WithContext(ctx))
		assert.Equal(http.StatusOK, response.Code)
		assert.False(gate.Open())
	}

	{
		var (
			response = httptest.NewRecorder()
			request  = httptest.NewRequest("POST", "/foo?open=true", nil)
		)

		lever.ServeHTTP(response, request.WithContext(ctx))
		assert.Equal(http.StatusCreated, response.Code)
		assert.True(gate.Open())
	}

	{
		var (
			response = httptest.NewRecorder()
			request  = httptest.NewRequest("POST", "/foo?open=false", nil)
		)

		lever.ServeHTTP(response, request.WithContext(ctx))
		assert.Equal(http.StatusCreated, response.Code)
		assert.False(gate.Open())
	}

	{
		var (
			response = httptest.NewRecorder()
			request  = httptest.NewRequest("POST", "/foo?open=false", nil)
		)

		lever.ServeHTTP(response, request.WithContext(ctx))
		assert.Equal(http.StatusOK, response.Code)
		assert.False(gate.Open())
	}
}

func TestLever(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("BadForm", testLeverServeHTTPBadForm)
		t.Run("NoParameter", testLeverServeHTTPNoParameter)
		t.Run("BadParameter", testLeverServeHTTPBadParameter)
		t.Run("Raise", testLeverServeHTTPRaise)
		t.Run("Lower", testLeverServeHTTPLower)
	})
}
