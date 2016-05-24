package handler

import (
	"bytes"
	"errors"
	"github.com/Comcast/webpa-common/fact"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"testing"
)

var httpErrorData = []struct {
	code    int
	message string
}{
	{http.StatusBadGateway, "random message"},
	{http.StatusOK, ""},
}

func TestNewHttpError(t *testing.T) {
	assert := assert.New(t)
	for _, record := range httpErrorData {
		actual := NewHttpError(record.code, record.message)
		assert.Equal(record.code, actual.Code())
		assert.Equal(record.message, actual.Error())
	}
}

func TestWriteJsonError(t *testing.T) {
	assert := assert.New(t)
	for _, record := range httpErrorData {
		response := httptest.NewRecorder()
		WriteJsonError(response, record.code, record.message)
		assertJsonErrorResponse(assert, response, record.code, record.message)
	}
}

func TestWriteErrorUsingError(t *testing.T) {
	assert := assert.New(t)
	for _, record := range httpErrorData {
		response := httptest.NewRecorder()
		err := errors.New(record.message)
		WriteError(response, err)
		assertJsonErrorResponse(assert, response, http.StatusInternalServerError, record.message)
	}
}

func TestWriteErrorUsingHttpError(t *testing.T) {
	assert := assert.New(t)
	for _, record := range httpErrorData {
		response := httptest.NewRecorder()
		WriteError(response, NewHttpError(record.code, record.message))
		assertJsonErrorResponse(assert, response, record.code, record.message)
	}
}

func TestWriteErrorUsingInt(t *testing.T) {
	assert := assert.New(t)
	response := httptest.NewRecorder()
	WriteError(response, http.StatusInternalServerError)
	assert.Equal(http.StatusInternalServerError, response.Code)
	assert.Equal(response.Header().Get(ContentTypeOptionsHeader), NoSniff)
	assert.Empty(response.Body.String())
}

func TestWriteErrorUsingString(t *testing.T) {
	assert := assert.New(t)
	const errorMessage string = "this is an error message"
	response := httptest.NewRecorder()
	WriteError(response, errorMessage)
	assertJsonErrorResponse(assert, response, http.StatusInternalServerError, errorMessage)
}

func TestWriteErrorUsingStringer(t *testing.T) {
	assert := assert.New(t)
	const errorMessage string = "this is an error message from a fmt.Stringer"
	response := httptest.NewRecorder()
	WriteError(response, bytes.NewBufferString(errorMessage))
	assertJsonErrorResponse(assert, response, http.StatusInternalServerError, errorMessage)
}

func TestRecoverFromPanic(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		panicValue         interface{}
		expectedStatusCode int
		expectedMessage    string
	}{
		{
			"an error message",
			http.StatusInternalServerError,
			"an error message",
		},
		{
			NewHttpError(415, "foobar!"),
			415,
			"foobar!",
		},
	}

	for _, record := range testData {
		ctx := context.Background()
		response := httptest.NewRecorder()

		// no logger in context ...
		func() {
			defer Recover(ctx, response)
			panic(record.panicValue)
		}()

		assertJsonErrorResponse(assert, response, record.expectedStatusCode, record.expectedMessage)

		var output bytes.Buffer
		logger := &logging.LoggerWriter{&output}
		ctx = fact.SetLogger(ctx, logger)
		response = httptest.NewRecorder()

		// now a logger is in the context
		func() {
			defer Recover(ctx, response)
			panic(record.panicValue)
		}()

		assertJsonErrorResponse(assert, response, record.expectedStatusCode, record.expectedMessage)
		if output.Len() == 0 {
			t.Error("Logger did not receive an error message")
		}
	}
}

func TestRecoverWithoutPanic(t *testing.T) {
	ctx := context.Background()
	response := httptest.NewRecorder()

	// no logger in context ...
	func() {
		defer Recover(ctx, response)
	}()

	if response.Code != 200 {
		t.Errorf("Unexpected status code: %d", response.Code)
	}

	if response.Body.Len() > 0 {
		t.Errorf("Unexpected response body: %s", response.Body.Bytes())
	}

	var output bytes.Buffer
	logger := &logging.LoggerWriter{&output}
	ctx = fact.SetLogger(ctx, logger)
	response = httptest.NewRecorder()

	// now a logger is in the context
	func() {
		defer Recover(ctx, response)
	}()

	if output.Len() > 0 {
		t.Error("Unexpected logging output")
	}

	if response.Code != 200 {
		t.Errorf("Unexpected status code: %d", response.Code)
	}

	if response.Body.Len() > 0 {
		t.Errorf("Unexpected response body: %s", response.Body.Bytes())
	}
}
