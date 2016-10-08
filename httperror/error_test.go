package httperror

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		expectedMessage string
		actualStatus    int
		expectedStatus  int
		expectedHeader  http.Header
	}{
		{
			expectedMessage: "this is a lovely message",
			actualStatus:    0,
			expectedStatus:  DefaultStatus,
			expectedHeader:  nil,
		},
		{
			expectedMessage: "here is another tasty message",
			actualStatus:    504,
			expectedStatus:  504,
			expectedHeader: http.Header{
				"Some-Header": []string{"Some Value"},
			},
		},
		{
			expectedMessage: "this message tastes like burning",
			actualStatus:    -1,
			expectedStatus:  DefaultStatus,
			expectedHeader: http.Header{
				"Some-Header":     []string{"Some Value"},
				"Multiple-Values": []string{"Value 1", "Value 2"},
			},
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)

		err := New(record.expectedMessage, record.actualStatus, record.expectedHeader)
		assert.NotNil(err)
		assert.Equal(record.expectedMessage, err.Error())
		assert.Equal(record.expectedMessage, err.String())
		assert.Equal(record.expectedStatus, err.Status())

		for expectedKey, expectedValues := range record.expectedHeader {
			assert.Equal(expectedValues, err.Header()[expectedKey])
		}
	}
}

// assertErrorResponse runs standard assertions on HTTP responses which have
// error output from this package
func assertErrorResponse(assert *assert.Assertions,
	response *httptest.ResponseRecorder,
	expectedMessage string,
	expectedStatus int,
	expectedHeader http.Header) {

	assert.Equal(expectedStatus, response.Code)
	for expectedKey, expectedValues := range expectedHeader {
		assert.Equal(expectedValues, response.HeaderMap[expectedKey])
	}

	assert.JSONEq(
		`{"message": "`+expectedMessage+`"}`,
		response.Body.String(),
	)
}

func TestWrite(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		expectedError  error
		expectedStatus int
		expectedHeader http.Header
	}{
		{
			expectedError:  errors.New("This is a standard golang error"),
			expectedStatus: DefaultStatus,
		},
		{
			expectedError:  New("Here is an HTTP error", 0, nil),
			expectedStatus: DefaultStatus,
		},
		{
			expectedError:  New("Here is an HTTP error", 513, http.Header{"Single-Value": []string{"a value"}}),
			expectedStatus: 513,
			expectedHeader: http.Header{"Single-Value": []string{"a value"}},
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		response := httptest.NewRecorder()

		count, writeError := Write(response, record.expectedError)
		assert.True(count > len(record.expectedError.Error()))
		assert.Nil(writeError)
		assertErrorResponse(assert, response, record.expectedError.Error(), record.expectedStatus, record.expectedHeader)
	}
}

func TestWriteMessage(t *testing.T) {
	assert := assert.New(t)
	response := httptest.NewRecorder()
	const expectedMessage = "here is a standalone message"

	count, writeError := WriteMessage(response, expectedMessage)
	assert.True(count > len(expectedMessage))
	assert.Nil(writeError)
	assertErrorResponse(assert, response, expectedMessage, DefaultStatus, nil)

}

func TestWriteFull(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		expectedMessage string
		actualStatus    int
		expectedStatus  int
		expectedHeader  http.Header
	}{
		{
			expectedMessage: "this is a simple message using all defaults",
			expectedStatus:  DefaultStatus,
		},
		{
			expectedMessage: "this is a message with a different status",
			actualStatus:    555,
			expectedStatus:  555,
		},
		{
			expectedMessage: "this is a message with everything supplied",
			actualStatus:    501,
			expectedStatus:  501,
			expectedHeader: http.Header{
				"Single-Value": []string{"a value"},
			},
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		response := httptest.NewRecorder()

		count, writeError := WriteFull(response, record.expectedMessage, record.actualStatus, record.expectedHeader)
		assert.True(count > len(record.expectedMessage))
		assert.Nil(writeError)
		assertErrorResponse(assert, response, record.expectedMessage, record.expectedStatus, record.expectedHeader)
	}
}
