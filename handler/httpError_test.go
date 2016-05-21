package handler

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
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
	for _, record := range httpErrorData {
		actual := NewHttpError(record.code, record.message)
		if record.code != actual.Code() {
			t.Errorf("Expected code %d, but got %d", record.code, actual.Code())
		}

		if record.message != actual.Error() {
			t.Errorf("Expected error message %s, but got %s", record.message, actual.Error())
		}
	}
}

func TestWriteJsonError(t *testing.T) {
	assertions := assert.New(t)
	for _, record := range httpErrorData {
		responseRecorder := httptest.NewRecorder()
		WriteJsonError(responseRecorder, record.code, record.message)

		if responseRecorder.Header().Get(ContentTypeHeader) != JsonContentType {
			t.Errorf("JSON content type not set")
		}

		if responseRecorder.Header().Get(ContentTypeOptionsHeader) != NoSniff {
			t.Errorf("nosniff content options not set")
		}

		if record.code != responseRecorder.Code {
			t.Errorf("Expected response code %d, but got %d", record.code, responseRecorder.Code)
		}

		assertions.JSONEq(
			fmt.Sprintf(`{"message": "%s"}`, record.message),
			responseRecorder.Body.String(),
		)
	}
}

func TestWriteErrorUsingHttpError(t *testing.T) {
	assertions := assert.New(t)
	for _, record := range httpErrorData {
		responseRecorder := httptest.NewRecorder()
		WriteError(responseRecorder, NewHttpError(record.code, record.message))

		if responseRecorder.Header().Get(ContentTypeHeader) != JsonContentType {
			t.Errorf("JSON content type not set")
		}

		if responseRecorder.Header().Get(ContentTypeOptionsHeader) != NoSniff {
			t.Errorf("nosniff content options not set")
		}

		if record.code != responseRecorder.Code {
			t.Errorf("Expected response code %d, but got %d", record.code, responseRecorder.Code)
		}

		assertions.JSONEq(
			fmt.Sprintf(`{"message": "%s"}`, record.message),
			responseRecorder.Body.String(),
		)
	}
}

func TestWriteErrorUsingInt(t *testing.T) {
	responseRecorder := httptest.NewRecorder()
	WriteError(responseRecorder, http.StatusInternalServerError)

	if responseRecorder.Header().Get(ContentTypeOptionsHeader) != NoSniff {
		t.Errorf("nosniff content options not set")
	}

	if http.StatusInternalServerError != responseRecorder.Code {
		t.Errorf("Expected response code %d, but got %d", http.StatusInternalServerError, responseRecorder.Code)
	}
}

func TestWriteErrorUsingString(t *testing.T) {
	assertions := assert.New(t)
	const errorMessage string = "this is an error message"
	responseRecorder := httptest.NewRecorder()
	WriteError(responseRecorder, errorMessage)

	if responseRecorder.Header().Get(ContentTypeHeader) != JsonContentType {
		t.Errorf("JSON content type not set")
	}

	if responseRecorder.Header().Get(ContentTypeOptionsHeader) != NoSniff {
		t.Errorf("nosniff content options not set")
	}

	if http.StatusInternalServerError != responseRecorder.Code {
		t.Errorf("Expected response code %d, but got %d", http.StatusInternalServerError, responseRecorder.Code)
	}

	assertions.JSONEq(
		fmt.Sprintf(`{"message": "%s"}`, errorMessage),
		responseRecorder.Body.String(),
	)
}

func TestWriteErrorUsingStringer(t *testing.T) {
	assertions := assert.New(t)
	const errorMessage string = "this is an error message from a fmt.Stringer"
	responseRecorder := httptest.NewRecorder()
	WriteError(responseRecorder, bytes.NewBufferString(errorMessage))

	if responseRecorder.Header().Get(ContentTypeHeader) != JsonContentType {
		t.Errorf("JSON content type not set")
	}

	if responseRecorder.Header().Get(ContentTypeOptionsHeader) != NoSniff {
		t.Errorf("nosniff content options not set")
	}

	if http.StatusInternalServerError != responseRecorder.Code {
		t.Errorf("Expected response code %d, but got %d", http.StatusInternalServerError, responseRecorder.Code)
	}

	assertions.JSONEq(
		fmt.Sprintf(`{"message": "%s"}`, errorMessage),
		responseRecorder.Body.String(),
	)
}
