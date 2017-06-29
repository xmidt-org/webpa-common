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

	// Invalid status value
	testHeader = http.Header{"X-Midt-Msg-Type": {"Auth"}, "X-Midt-Status": {"Invalid"}}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)

	// Mandatory status header
	testHeader = http.Header{"X-Midt-Msg-Type": {"Auth"}}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Status header string", err.Error())

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
		"X-Midt-Msg-Type":                 {"SimpleRequestResponse"},
		"X-Midt-Source":                   {"test"},
		"X-Midt-Content-Type":             {"application/json"},
		"X-Midt-Accept":                   {"application/json"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":            {"true"},
		"X-Midt-Spans":                    {"client1", "14678987563", "200", "client2", "146564565673", "500"},
		"X-Midt-Request-Delivery-Reponse": {"1234"},
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
	assert.Equal(int64(1234), *msg.RequestDeliveryResponse)

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

func TestHeaderToWRP_Create(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":                 {"Create"},
		"X-Midt-Source":                   {"src"},
		"X-Midt-Path":                     {"/webpa-uuid"},
		"X-Midt-Content-Type":             {"application/json"},
		"X-Midt-Accept":                   {"application/json"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2"},
		"X-Midt-Request-Delivery-Reponse": {"534290"},
		"X-Midt-Include-Spans":            {"true"},
		"X-Midt-Spans":                    {"client1", "14678987563", "200"},
	}
	msg, err := HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(CreateMessageType, msg.MessageType())
	assert.Equal("src", msg.Source)
	assert.Equal("test_transaction_id01", msg.TransactionUUID)
	assert.Equal("/webpa-uuid", msg.Path)
	assert.Equal([]string{"key1", "value1", "key2", "value2"}, msg.Headers)
	assert.Equal(true, *msg.IncludeSpans)
	assert.Equal([]string{"client1", "14678987563", "200"}, msg.Spans[0])
	assert.Equal(int64(534290), *msg.RequestDeliveryResponse)

	// Invalid MsgType and Source and Transaction uuid
	testHeader = http.Header{
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
	assert.Equal("Invalid Message Type header string", err.Error())

	// Invalid Transaction_uuid
	testHeader = http.Header{
		"X-Midt-Msg-Type":      {"Create"},
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
		"X-Midt-Msg-Type":         {"Create"},
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

	// Invalid Path
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"Create"},
		"X-Midt-Source":           {"test"},
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
	assert.Equal("Invalid Path header string", err.Error())
}

func TestHeaderToWRP_Retrieve(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":         {"Retrieve"},
		"X-Midt-Source":           {"src"},
		"X-Midt-Path":             {"/webpa-uuid"},
		"X-Midt-Content-Type":     {"application/json"},
		"X-Midt-Accept":           {"application/json"},
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Headers":          {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":    {"true"},
		"X-Midt-Spans":            {"client1", "14678987563", "200"},
	}
	msg, err := HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(RetrieveMessageType, msg.MessageType())
	assert.Equal("src", msg.Source)
	assert.Equal("test_transaction_id01", msg.TransactionUUID)
	assert.Equal("/webpa-uuid", msg.Path)
	assert.Equal([]string{"key1", "value1", "key2", "value2"}, msg.Headers)
	assert.Equal(true, *msg.IncludeSpans)
	assert.Equal([]string{"client1", "14678987563", "200"}, msg.Spans[0])

	// Invalid MsgType and Source and Transaction uuid
	testHeader = http.Header{
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
	assert.Equal("Invalid Message Type header string", err.Error())

	// Invalid Transaction_uuid
	testHeader = http.Header{
		"X-Midt-Msg-Type":      {"Retrieve"},
		"X-Midt-Source":        {"test"},
		"X-Midt-Content-Type":  {"application/json"},
		"X-Midt-Accept":        {"application/json"},
		"X-Midt-Headers":       {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans": {"true"},
		"X-Midt-Spans":         {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Transaction_Uuid header string", err.Error())

	// Invalid Source
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"Retrieve"},
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

	// Invalid Path
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"Retrieve"},
		"X-Midt-Source":           {"test"},
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
	assert.Equal("Invalid Path header string", err.Error())
}

func TestHeaderToWRP_Update(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":         {"Update"},
		"X-Midt-Source":           {"src"},
		"X-Midt-Path":             {"/webpa-uuid"},
		"X-Midt-Content-Type":     {"application/json"},
		"X-Midt-Accept":           {"application/json"},
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Headers":          {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":    {"true"},
		"X-Midt-Spans":            {"client1", "14678987563", "200"},
	}
	msg, err := HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(UpdateMessageType, msg.MessageType())
	assert.Equal("src", msg.Source)
	assert.Equal("test_transaction_id01", msg.TransactionUUID)
	assert.Equal("/webpa-uuid", msg.Path)
	assert.Equal([]string{"key1", "value1", "key2", "value2"}, msg.Headers)
	assert.Equal(true, *msg.IncludeSpans)
	assert.Equal([]string{"client1", "14678987563", "200"}, msg.Spans[0])

	// Invalid MsgType and Source and Transaction uuid
	testHeader = http.Header{
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
	assert.Equal("Invalid Message Type header string", err.Error())

	// Invalid Transaction_uuid
	testHeader = http.Header{
		"X-Midt-Msg-Type":      {"Update"},
		"X-Midt-Source":        {"test"},
		"X-Midt-Content-Type":  {"application/json"},
		"X-Midt-Accept":        {"application/json"},
		"X-Midt-Headers":       {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans": {"true"},
		"X-Midt-Spans":         {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Transaction_Uuid header string", err.Error())

	// Invalid Source
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"Update"},
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

	// Invalid Path
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"Update"},
		"X-Midt-Source":           {"test"},
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
	assert.Equal("Invalid Path header string", err.Error())
}

func TestHeaderToWRP_Delete(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":         {"Delete"},
		"X-Midt-Source":           {"src"},
		"X-Midt-Path":             {"/webpa-uuid"},
		"X-Midt-Content-Type":     {"application/json"},
		"X-Midt-Accept":           {"application/json"},
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Headers":          {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":    {"true"},
		"X-Midt-Spans":            {"client1"},
	}
	msg, err := HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(DeleteMessageType, msg.MessageType())
	assert.Equal("src", msg.Source)
	assert.Equal("test_transaction_id01", msg.TransactionUUID)
	assert.Equal("/webpa-uuid", msg.Path)
	assert.Equal([]string{"key1", "value1", "key2", "value2"}, msg.Headers)
	assert.Equal(true, *msg.IncludeSpans)
	assert.Equal([]string{"client1"}, msg.Spans[0])

	// Invalid MsgType and Source and Transaction uuid
	testHeader = http.Header{
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
	assert.Equal("Invalid Message Type header string", err.Error())

	// Invalid Transaction_uuid
	testHeader = http.Header{
		"X-Midt-Msg-Type":      {"Delete"},
		"X-Midt-Source":        {"test"},
		"X-Midt-Content-Type":  {"application/json"},
		"X-Midt-Accept":        {"application/json"},
		"X-Midt-Headers":       {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans": {"true"},
		"X-Midt-Spans":         {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Transaction_Uuid header string", err.Error())

	// Invalid Source
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"Delete"},
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

	// Invalid Path
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"Delete"},
		"X-Midt-Source":           {"test"},
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
	assert.Equal("Invalid Path header string", err.Error())
}
