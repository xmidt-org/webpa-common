package wrp

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestStringToMessageType(t *testing.T) {
	assert := assert.New(t)
	testStrArr := []string{
		"Auth",
		"SimpleRequestResponse",
		"SimpleEvent",
		"Create",
		"Retrieve",
		"Update",
		"Delete",
		"ServiceRegistration",
		"ServiceAlive",
		"Invalid",
		"TestMsg",
		"",
	}
	expectedMsgType := []MessageType{
		AuthMessageType,
		SimpleRequestResponseMessageType,
		SimpleEventMessageType,
		CreateMessageType,
		RetrieveMessageType,
		UpdateMessageType,
		DeleteMessageType,
		ServiceRegistrationMessageType,
		ServiceAliveMessageType,
		MessageType(-1),
		MessageType(-1),
		MessageType(-1),
	}

	for i, str := range testStrArr {
		msgType := StringToMessageType(str)
		assert.Equal(expectedMsgType[i], msgType)
	}
}

func TestHeaderToWRP_Auth(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{"X-Midt-Msg-Type": {"Auth"}, "X-Midt-Status": {"200"}}
	msg, err := HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(msg.MessageType(), AuthMessageType)
	assert.Equal(int64(200), *msg.Status)

	// Invalid status
	testHeader = http.Header{"X-Midt-Msg-Type": {"Auth"}, "X-Midt-Status": {"Invalid"}}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)

	// Invalid MsgType
	testHeader = http.Header{"X-Midt-Msg-Type": {"Invalid"}, "X-Midt-Status": {"400"}}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Message Type header string", err.Error())
}

func TestHeaderToWRP_SimpleRequest(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":         {"SimpleRequestResponse"},
		"X-Midt-Source":           {"test"},
		"X-Midt-Content-Type":     {"application/json"},
		"X-Midt-Accept":           {"application/json"},
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Headers":          {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":    {"true"},
		"X-Midt-Spans":            {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	msg, err := HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(SimpleRequestResponseMessageType, msg.MessageType())
	assert.Equal("test", msg.Source)
	assert.Equal("test_transaction_id01", msg.TransactionUUID)
	assert.Equal([]string{"key1", "value1", "key2", "value2"}, msg.Headers)
	assert.Equal(true, *msg.IncludeSpans)
	assert.Equal([]string{"client1", "14678987563", "200"}, msg.Spans[0])
	assert.Equal([]string{"client2", "146564565673", "500"}, msg.Spans[1])

	// Invalid MsgType
	testHeader = http.Header{
		"X-Midt-Source":           {"test"},
		"X-Midt-Content-Type":     {"application/json"},
		"X-Midt-Accept":           {"application/json"},
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Headers":          {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":    {"true"},
		"X-Midt-Spans":            {"client1", "14678987563", "200", "client2", "146564565673", "500"},
		"X-Midt-Msg-Type":         {"Invalid"}, "X-Midt-Status": {"400"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Message Type header string", err.Error())

	// Invalid Transaction_uuid
	testHeader = http.Header{
		"X-Midt-Msg-Type":      {"SimpleRequestResponse"},
		"X-Midt-Source":        {"test"},
		"X-Midt-Content-Type":  {"application/json"},
		"X-Midt-Accept":        {"application/json"},
		"X-Midt-Headers":       {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans": {"true"},
		"X-Midt-Spans":         {"client1", "14678987563", "200", "client2", "146564565673", "500"},
		"X-Midt-Status":        {"400"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Transaction_Uuid header string", err.Error())

	// Invalid Source
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"SimpleRequestResponse"},
		"X-Midt-Content-Type":     {"application/json"},
		"X-Midt-Accept":           {"application/json"},
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Headers":          {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":    {"true"},
		"X-Midt-Spans":            {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Source header string", err.Error())
}

func TestHeaderToWRP_SimpleEvent(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":     {"SimpleEvent"},
		"X-Midt-Source":       {"test"},
		"X-Midt-Content-Type": {"application/json"},
		"X-Midt-Accept":       {"application/json"},
		"X-Midt-Headers":      {"key1", "value1", "key2"},
	}
	msg, err := HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(SimpleEventMessageType, msg.MessageType())
	assert.Equal("test", msg.Source)
	assert.Equal([]string{"key1", "value1", "key2"}, msg.Headers)
	assert.Equal("application/json", msg.ContentType)

	// Invalid MsgType
	testHeader = http.Header{
		"X-Midt-Source":  {"test"},
		"X-Midt-Headers": {"key1", "value1", "key2"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Message Type header string", err.Error())

	// Invalid Source
	testHeader = http.Header{
		"X-Midt-Msg-Type": {"SimpleEvent"},
		"X-Midt-Headers":  {"key1", "value1", "key2"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Source header string", err.Error())
}
