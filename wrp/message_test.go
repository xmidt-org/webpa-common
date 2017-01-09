package wrp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessageTypeString(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		messageType    MessageType
		expectedString string
	}{
		{MessageType(0), invalidMessageTypeString},
		{MessageType(1), invalidMessageTypeString},
		{AuthMessageType, messageTypeStrings[AuthMessageType]},
		{SimpleRequestResponseMessageType, messageTypeStrings[SimpleRequestResponseMessageType]},
		{SimpleEventMessageType, messageTypeStrings[SimpleEventMessageType]},
		{CRUDCreateMessageType, messageTypeStrings[CRUDCreateMessageType]},
		{CRUDRetrieveMessageType, messageTypeStrings[CRUDRetrieveMessageType]},
		{CRUDUpdateMessageType, messageTypeStrings[CRUDUpdateMessageType]},
		{CRUDDeleteMessageType, messageTypeStrings[CRUDDeleteMessageType]},
		{MessageType(999), invalidMessageTypeString},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		assert.Equal(record.expectedString, record.messageType.String())
	}
}

func TestMessageValid(t *testing.T) {
	assert := assert.New(t)
	expectedStatus := int64(987)
	expectedUUID := "transId12345"
	expectedSource := "mac:112233445566"
	expectedDestination := "dns:foobar.com"
	expectedPath := "/path/to/crud"
	expectedPayload := []byte("the package")

	var testData = []struct {
		message       Message
		expectedValid bool
	}{
		{
			Message{},
			false,
		},
		{
			Message{
				Type: MessageType(12345),
			},
			false,
		},
		{
			Message{
				Type: AuthMessageType,
			},
			false,
		},
		{
			Message{
				Type:   AuthMessageType,
				Status: &expectedStatus,
			},
			true,
		},
		{
			Message{
				Type: SimpleRequestResponseMessageType,
			},
			false,
		},
		{
			Message{
				Type:            SimpleRequestResponseMessageType,
				TransactionUUID: expectedUUID,
				Source:          expectedSource,
				Destination:     expectedDestination,
				Payload:         expectedPayload,
			},
			true,
		},
		{
			Message{
				Type: SimpleEventMessageType,
			},
			false,
		},
		{
			Message{
				Type:        SimpleEventMessageType,
				Source:      expectedSource,
				Destination: expectedDestination,
				Payload:     expectedPayload,
			},
			true,
		},
		{
			Message{
				Type: CRUDCreateMessageType,
			},
			false,
		},
		{
			Message{
				Type:        CRUDCreateMessageType,
				Source:      expectedSource,
				Destination: expectedDestination,
				Path:        expectedPath,
			},
			true,
		},
		{
			Message{
				Type: CRUDRetrieveMessageType,
			},
			false,
		},
		{
			Message{
				Type:        CRUDRetrieveMessageType,
				Source:      expectedSource,
				Destination: expectedDestination,
				Path:        expectedPath,
			},
			true,
		},
		{
			Message{
				Type: CRUDUpdateMessageType,
			},
			false,
		},
		{
			Message{
				Type:        CRUDUpdateMessageType,
				Source:      expectedSource,
				Destination: expectedDestination,
				Path:        expectedPath,
			},
			true,
		},
		{
			Message{
				Type: CRUDDeleteMessageType,
			},
			false,
		},
		{
			Message{
				Type:        CRUDDeleteMessageType,
				Source:      expectedSource,
				Destination: expectedDestination,
				Path:        expectedPath,
			},
			true,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		assert.Equal(record.expectedValid, record.message.Valid() == nil)
	}
}

func TestMessageNilString(t *testing.T) {
	assert := assert.New(t)
	var message *Message
	assert.Equal("nil", message.String())
}

func TestMessageDeduceType(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		original     Message
		expectedType MessageType
		expectsError bool
	}{
		{
			Message{},
			MessageType(0),
			true,
		},
		{
			Message{Type: MessageType(999)},
			MessageType(999),
			true,
		},
		{
			Message{
				Status: &expectedStatus,
			},
			AuthMessageType,
			false,
		},
		{
			Message{
				Source:      "serial:1234",
				Destination: "foobar.com",
				Payload:     []byte("someone is here..."),
			},
			SimpleEventMessageType,
			false,
		},
		{
			Message{
				TransactionUUID: "transId12345",
				Source:          "serial:1234",
				Destination:     "foobar.com",
				Payload:         []byte("i see you."),
			},
			SimpleRequestResponseMessageType,
			false,
		},
		{
			Message{
				Type:         MessageType(5),  // todo: this should be removed once we can tell the difference between crud types
				Source:       "serial:1234",
				Destination:  "foobar.com",
				Path:         "/path/to/create",
			},
			CRUDCreateMessageType,
			true,  // todo: this should be changed to false one we can tell the difference between crud types
		},
		{
			Message{
				Type:         MessageType(6),  // todo: this should be removed once we can tell the difference between crud types
				Source:       "serial:1234",
				Destination:  "foobar.com",
				Path:         "/path/to/retrieve",
			},
			CRUDRetrieveMessageType,
			true,  // todo: this should be changed to false one we can tell the difference between crud types
		},
		{
			Message{
				Type:         MessageType(7),  // todo: this should be removed once we can tell the difference between crud types
				Source:       "serial:1234",
				Destination:  "foobar.com",
				Path:         "/path/to/update",
			},
			CRUDUpdateMessageType,
			true,  // todo: this should be changed to false one we can tell the difference between crud types
		},
		{
			Message{
				Type:         MessageType(8),  // todo: this should be removed once we can tell the difference between crud types
				Source:       "serial:1234",
				Destination:  "foobar.com",
				Path:         "/path/to/delete",
			},
			CRUDDeleteMessageType,
			true,  // todo: this should be changed to false one we can tell the difference between crud types
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)

		message := record.original
		assert.Equal(record.expectsError, message.DeduceType() != nil)
		assert.Equal(record.expectedType, message.Type)
	}
}

func TestNewAuth(t *testing.T) {
	assert := assert.New(t)

	for _, expectedStatus := range []int64{-1234, 0, 1, 2, 239457} {
		t.Logf("%d", expectedStatus)
		message := NewAuth(expectedStatus)
		if assert.NotNil(message) {
			assert.Equal(AuthMessageType, message.Type)
			assert.Empty(message.TransactionUUID)
			assert.Empty(message.Source)
			assert.Empty(message.Destination)
			assert.Equal(expectedStatus, *message.Status)
			assert.Empty(message.Payload)
		}
	}
}

func TestNewSimpleRequestResponse(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		expectedDestination     string
		expectedSource          string
		expectedTransactionUUID string
		expectedPayload         []byte
	}{
		{"foobar.com", "112233445566", "transId12345", nil},
		{"test.com/bleh", "FFFFEEEEDDDD", "transId54321", []byte("hi there!")},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		message := NewSimpleRequestResponse(record.expectedDestination, record.expectedSource, record.expectedTransactionUUID, record.expectedPayload)
		if assert.NotNil(message) {
			assert.Equal(SimpleRequestResponseMessageType, message.Type)
			assert.Equal(record.expectedTransactionUUID, message.TransactionUUID)
			assert.Equal(record.expectedSource, message.Source)
			assert.Equal(record.expectedDestination, message.Destination)
			assert.Empty(message.Headers)
			assert.Empty(message.Metadata)
			assert.Empty(message.Spans)
			assert.Empty(message.IncludeSpans)
			assert.Nil(message.Status)
			assert.Empty(message.Path)
			assert.Equal(record.expectedPayload, message.Payload)
		}
	}
}

func TestNewSimpleEvent(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		expectedDestination string
		expectedSource      string
		expectedPayload     []byte
	}{
		{"foobar.com", "112233445566", nil},
		{"test.com/bleh", "FFFFEEEEDDDD", []byte("hi there!")},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		message := NewSimpleEvent(record.expectedDestination, record.expectedSource, record.expectedPayload)
		if assert.NotNil(message) {
			assert.Equal(SimpleEventMessageType, message.Type)
			assert.Empty(message.TransactionUUID)
			assert.Equal(record.expectedSource, message.Source)
			assert.Equal(record.expectedDestination, message.Destination)
			assert.Empty(message.Headers)
			assert.Empty(message.Metadata)
			assert.Empty(message.Spans)
			assert.Empty(message.IncludeSpans)
			assert.Nil(message.Status)
			assert.Empty(message.Path)
			assert.Equal(record.expectedPayload, message.Payload)
		}
	}
}

// assertCRUDMessageType validates the results of a CRUD message type against what is expected.
func assertCRUDMessageType(assert *assert.Assertions, expected crudTestData, actual *Message) {
	if assert.NotNil(actual) {
		assert.Equal(expected.expectedMessageType, actual.Type)
		assert.Empty(actual.TransactionUUID)
		assert.Equal(expected.expectedSource, actual.Source)
		assert.Equal(expected.expectedDestination, actual.Destination)
		assert.Empty(actual.Headers)
		assert.Empty(actual.Metadata)
		assert.Empty(actual.Spans)
		assert.Empty(actual.IncludeSpans)
		assert.Nil(actual.Status)
		assert.Equal(expected.expectedPath, actual.Path)
		assert.Empty(actual.Payload)
	}
}

type crudTestData struct {
	expectedMessageType MessageType
	expectedDestination string
	expectedSource      string
	expectedPath        string
}

func TestNewCRUD(t *testing.T) {
	assert := assert.New(t)
	var testData = []crudTestData{
		{CRUDCreateMessageType, "foobar.com", "112233445566", "/path/to/create"},
		{CRUDRetrieveMessageType, "test.com/bleh", "FFFFEEEEDDDD", "/path/to/retrieve"},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		message := newCRUD(record.expectedMessageType, record.expectedDestination, record.expectedSource, record.expectedPath)
		assertCRUDMessageType(assert, record, message)
	}
}

func TestNewCRUDCreate(t *testing.T) {
	assert := assert.New(t)
	var testData = []crudTestData{
		{CRUDCreateMessageType, "foobar.com", "112233445566", ""},
		{CRUDCreateMessageType, "test.com/bleh", "FFFFEEEEDDDD", "/path/to/create"},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		message := NewCRUDCreate(record.expectedDestination, record.expectedSource, record.expectedPath)
		assertCRUDMessageType(assert, record, message)
	}
}

func TestNewCRUDRetrieve(t *testing.T) {
	assert := assert.New(t)
	var testData = []crudTestData{
		{CRUDRetrieveMessageType, "foobar.com", "112233445566", ""},
		{CRUDRetrieveMessageType, "test.com/bleh", "FFFFEEEEDDDD", "/path/to/retrieve"},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		message := NewCRUDRetrieve(record.expectedDestination, record.expectedSource, record.expectedPath)
		assertCRUDMessageType(assert, record, message)
	}
}

func TestNewCRUDUpdate(t *testing.T) {
	assert := assert.New(t)
	var testData = []crudTestData{
		{CRUDUpdateMessageType, "foobar.com", "112233445566", ""},
		{CRUDUpdateMessageType, "test.com/bleh", "FFFFEEEEDDDD", "/path/to/update"},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		message := NewCRUDUpdate(record.expectedDestination, record.expectedSource, record.expectedPath)
		assertCRUDMessageType(assert, record, message)
	}
}

func TestNewCRUDDelete(t *testing.T) {
	assert := assert.New(t)
	var testData = []crudTestData{
		{CRUDDeleteMessageType, "foobar.com", "112233445566", ""},
		{CRUDDeleteMessageType, "test.com/bleh", "FFFFEEEEDDDD", "/path/to/delete"},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		message := NewCRUDDelete(record.expectedDestination, record.expectedSource, record.expectedPath)
		assertCRUDMessageType(assert, record, message)
	}
}