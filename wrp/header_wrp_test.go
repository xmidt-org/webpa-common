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

	// Missing status header => success
	testHeader = http.Header{"X-Midt-Msg-Type": {"Auth"}}
	msg, err = HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(msg.MessageType(), AuthMessageType)

	// Invalid MsgType
	testHeader = http.Header{"X-Midt-Msg-Type": {"Invalid"}, "X-Midt-Status": {"400"}}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal(ErrInvalidMsgType, err)
}

func TestHeaderToWRP_SimpleRequest(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":                 {"SimpleRequestResponse"},
		"X-Midt-Source":                   {"test"},
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
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Headers":          {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":    {"true"},
		"X-Midt-Spans":            {"client1", "14678987563", "200", "client2", "146564565673", "500"},
		"X-Midt-Msg-Type":         {"Invalid"}, "X-Midt-Status": {"400"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal(ErrInvalidMsgType, err)

	// Missing Transaction_uuid
	testHeader = http.Header{
		"X-Midt-Msg-Type":      {"SimpleRequestResponse"},
		"X-Midt-Source":        {"test"},
		"X-Midt-Headers":       {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans": {"true"},
		"X-Midt-Spans":         {"client1", "14678987563", "200", "client2", "146564565673", "500"},
		"X-Midt-Status":        {"400"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(SimpleRequestResponseMessageType, msg.MessageType())
	assert.Equal("test", msg.Source)
	assert.Equal([]string{"key1", "value1", "key2", "value2"}, msg.Headers)
	assert.Equal(true, *msg.IncludeSpans)
	assert.Equal([]string{"client1", "14678987563", "200"}, msg.Spans[0])
	assert.Equal([]string{"client2", "146564565673", "500"}, msg.Spans[1])
	assert.Equal(int64(400), *msg.Status)

	// Missing Source
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"SimpleRequestResponse"},
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Headers":          {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":    {"true"},
		"X-Midt-Spans":            {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(SimpleRequestResponseMessageType, msg.MessageType())
	assert.Equal("test_transaction_id01", msg.TransactionUUID)
	assert.Equal([]string{"key1", "value1", "key2", "value2"}, msg.Headers)
	assert.Equal(true, *msg.IncludeSpans)
	assert.Equal([]string{"client1", "14678987563", "200"}, msg.Spans[0])
	assert.Equal([]string{"client2", "146564565673", "500"}, msg.Spans[1])

	// Invalid RDR
	testHeader = http.Header{
		"X-Midt-Msg-Type":                 {"SimpleRequestResponse"},
		"X-Midt-Source":                   {"test"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Request-Delivery-Reponse": {"Invalid"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
}

func TestHeaderToWRP_SimpleEvent(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type": {"SimpleEvent"},
		"X-Midt-Source":   {"test"},
		"X-Midt-Headers":  {"key1", "value1", "key2"},
	}
	msg, err := HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(SimpleEventMessageType, msg.MessageType())
	assert.Equal("test", msg.Source)
	assert.Equal([]string{"key1", "value1", "key2"}, msg.Headers)

	// Missing MsgType
	testHeader = http.Header{
		"X-Midt-Source":  {"test"},
		"X-Midt-Headers": {"key1", "value1", "key2"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal(ErrInvalidMsgType, err)

	// Missing Source
	testHeader = http.Header{
		"X-Midt-Msg-Type": {"SimpleEvent"},
		"X-Midt-Headers":  {"key1", "value1", "key2"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(SimpleEventMessageType, msg.MessageType())
	assert.Equal([]string{"key1", "value1", "key2"}, msg.Headers)
}

func TestHeaderToWRP_Create(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":                 {"Create"},
		"X-Midt-Source":                   {"src"},
		"X-Midt-Path":                     {"/webpa-uuid"},
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

	// Missing MsgType and Source and Transaction uuid
	testHeader = http.Header{
		"X-Midt-Headers":       {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans": {"true"},
		"X-Midt-Spans":         {"client1", "14678987563", "200", "client2", "146564565673", "500"},
		"X-Midt-Status":        {"400"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal(ErrInvalidMsgType, err)

	// Missing Transaction_uuid
	testHeader = http.Header{
		"X-Midt-Msg-Type":                 {"Create"},
		"X-Midt-Source":                   {"test"},
		"X-Midt-Path":                     {"/webpa-uuid"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":            {"true"},
		"X-Midt-Spans":                    {"client1", "14678987563", "200", "client2", "146564565673", "500"},
		"X-Midt-Request-Delivery-Reponse": {"400"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(CreateMessageType, msg.MessageType())
	assert.Equal("test", msg.Source)
	assert.Equal("/webpa-uuid", msg.Path)
	assert.Equal([]string{"key1", "value1", "key2", "value2"}, msg.Headers)
	assert.Equal(true, *msg.IncludeSpans)
	assert.Equal([]string{"client1", "14678987563", "200"}, msg.Spans[0])
	assert.Equal(int64(400), *msg.RequestDeliveryResponse)

	// Missing Source
	testHeader = http.Header{
		"X-Midt-Msg-Type":                 {"Create"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Path":                     {"/webpa-uuid"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":            {"false"},
		"X-Midt-Spans":                    {"client1", "14678987563", "200", "client2", "146564565673", "500"},
		"X-Midt-Request-Delivery-Reponse": {"400"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(CreateMessageType, msg.MessageType())
	assert.Equal("test_transaction_id01", msg.TransactionUUID)
	assert.Equal("/webpa-uuid", msg.Path)
	assert.Equal([]string{"key1", "value1", "key2", "value2"}, msg.Headers)
	assert.Equal(false, *msg.IncludeSpans)
	assert.Equal([]string{"client1", "14678987563", "200"}, msg.Spans[0])
	assert.Equal(int64(400), *msg.RequestDeliveryResponse)

	// Missing Path
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"Create"},
		"X-Midt-Source":           {"test"},
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Headers":          {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":    {"true"},
		"X-Midt-Spans":            {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(CreateMessageType, msg.MessageType())
	assert.Equal("test", msg.Source)
	assert.Equal("test_transaction_id01", msg.TransactionUUID)
	assert.Equal([]string{"key1", "value1", "key2", "value2"}, msg.Headers)
	assert.Equal(true, *msg.IncludeSpans)
	assert.Equal([]string{"client1", "14678987563", "200"}, msg.Spans[0])

	// Invalid RDR
	testHeader = http.Header{
		"X-Midt-Msg-Type":                 {"Create"},
		"X-Midt-Source":                   {"src"},
		"X-Midt-Path":                     {"/webpa-uuid"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2"},
		"X-Midt-Request-Delivery-Reponse": {"Invalid"},
		"X-Midt-Include-Spans":            {"true"},
		"X-Midt-Spans":                    {"client1", "14678987563", "200"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)

	// Invalid bool
	testHeader = http.Header{
		"X-Midt-Msg-Type":                 {"Create"},
		"X-Midt-Source":                   {"src"},
		"X-Midt-Path":                     {"/webpa-uuid"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2"},
		"X-Midt-Request-Delivery-Reponse": {"1234"},
		"X-Midt-Include-Spans":            {"Invalid"},
		"X-Midt-Spans":                    {"client1", "14678987563", "200"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
}

func TestHeaderToWRP_Retrieve(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":         {"Retrieve"},
		"X-Midt-Source":           {"src"},
		"X-Midt-Path":             {"/webpa-uuid"},
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

	// Missing Path
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"Retrieve"},
		"X-Midt-Source":           {"test"},
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Include-Spans":    {"false"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(RetrieveMessageType, msg.MessageType())
	assert.Equal("test", msg.Source)
	assert.Equal("test_transaction_id01", msg.TransactionUUID)
	assert.Equal(false, *msg.IncludeSpans)
}

func TestHeaderToWRP_Update(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":         {"Update"},
		"X-Midt-Source":           {"src"},
		"X-Midt-Path":             {"/webpa-uuid"},
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

}

func TestHeaderToWRP_Delete(t *testing.T) {
	assert := assert.New(t)

	// Success case
	testHeader := http.Header{
		"X-Midt-Msg-Type":         {"Delete"},
		"X-Midt-Source":           {"src"},
		"X-Midt-Path":             {"/webpa-uuid"},
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

	// Invalid bool
	testHeader = http.Header{
		"X-Midt-Msg-Type":         {"Delete"},
		"X-Midt-Source":           {"src"},
		"X-Midt-Transaction-Uuid": {"test_transaction_id01"},
		"X-Midt-Headers":          {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":    {"abcd"},
		"X-Midt-Spans":            {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	msg, err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
}

func TestWRPToHeader_Auth(t *testing.T) {
	assert := assert.New(t)

	status := int64(200)
	// Success case
	expectedHeader := http.Header{"X-Midt-Msg-Type": {"Auth"}, "X-Midt-Status": {"200"}}
	msg := Message{Type: AuthMessageType, Status: &status}

	header, err := WRPToHeader(&msg)
	assert.Nil(err)
	assert.Equal("Auth", header.Get(MsgTypeHeader))
	assert.Equal("200", header.Get(StatusHeader))
	assert.Equal(expectedHeader, header)

	// Invalid MessageType
	status = int64(123)
	msg = Message{Type: MessageType(-1), Status: &status}
	header, err = WRPToHeader(&msg)
	assert.Nil(header)
	assert.Equal(ErrInvalidMsgType, err)

	// Missing status header
	expectedHeader = http.Header{"X-Midt-Msg-Type": {"Auth"}}
	msg = Message{Type: AuthMessageType}
	header, err = WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	assert.Equal(expectedHeader, header)

	// Invalid status
	status = int64(0)
	msg = Message{Type: AuthMessageType, Status: &status}
	header, err = WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	expectedHeader = http.Header{"X-Midt-Msg-Type": {"Auth"}, "X-Midt-Status": {"0"}}
	assert.Equal(expectedHeader, header)
}

func TestWRPToHeader_SimpleRequest(t *testing.T) {
	assert := assert.New(t)

	// Success case
	expectedHeader := http.Header{
		"X-Midt-Msg-Type":                 {"SimpleRequestResponse"},
		"X-Midt-Source":                   {"test"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2"},
		"X-Midt-Include-Spans":            {"true"},
		"X-Midt-Spans":                    {"client1", "14678987563", "200", "client2", "146564565673", "500"},
		"X-Midt-Request-Delivery-Reponse": {"1234"},
	}

	rdr := int64(1234)
	incSpan := true
	msg := Message{Type: SimpleRequestResponseMessageType,
		Source:          "test",
		TransactionUUID: "test_transaction_id01",
		Headers:         []string{"key1", "value1", "key2", "value2"},
		Spans: [][]string{{"client1", "14678987563", "200"},
			{"client2", "146564565673", "500"}},
	}
	msg.RequestDeliveryResponse = &rdr
	msg.IncludeSpans = &incSpan

	header, err := WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	assert.Equal(expectedHeader, header)

	// Invalid MsgType
	msg = Message{
		Source:          "test",
		TransactionUUID: "test_transaction_id01",
		Headers:         []string{"key1", "value1", "key2", "value2"},
		Spans: [][]string{{"client1", "14678987563", "200"},
			{"client2", "146564565673", "500"}},
	}
	header, err = WRPToHeader(&msg)
	assert.Nil(header)
	assert.NotNil(err)
	assert.Equal(ErrInvalidMsgType, err)

	// Missing Transaction_uuid
	msg = Message{Type: SimpleRequestResponseMessageType,
		Source:  "test",
		Headers: []string{"key1", "value1", "key2", "value2"},
		Spans: [][]string{{"client1", "14678987563", "200"},
			{"client2", "146564565673", "500"}},
	}
	expectedHeader = http.Header{
		"X-Midt-Msg-Type": {"SimpleRequestResponse"},
		"X-Midt-Source":   {"test"},
		"X-Midt-Headers":  {"key1", "value1", "key2", "value2"},
		"X-Midt-Spans":    {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	header, err = WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	assert.Equal(expectedHeader, header)
}

func TestWRPToHeader_SimpleEvent(t *testing.T) {
	assert := assert.New(t)

	// Success case
	expectedHeader := http.Header{
		"X-Midt-Msg-Type": {"SimpleEvent"},
		"X-Midt-Source":   {"test"},
		"X-Midt-Headers":  {"key1", "value1", "key2"},
	}
	msg := Message{Type: SimpleEventMessageType,
		Source:  "test",
		Headers: []string{"key1", "value1", "key2"},
	}
	header, err := WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	assert.Equal(expectedHeader, header)

	// Invalid MsgType
	msg = Message{Type: MessageType(-10),
		Source:  "test",
		Headers: []string{"key1", "value1", "key2", "value2"},
	}
	header, err = WRPToHeader(&msg)
	assert.Nil(header)
	assert.NotNil(err)
	assert.Equal(ErrInvalidMsgType, err)
}

func TestWRPToHeader_Create(t *testing.T) {
	assert := assert.New(t)

	// Success case
	expectedHeader := http.Header{
		"X-Midt-Msg-Type":                 {"Create"},
		"X-Midt-Source":                   {"src"},
		"X-Midt-Path":                     {"/webpa-uuid"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2", "key3", "value3"},
		"X-Midt-Request-Delivery-Reponse": {"534290"},
		"X-Midt-Include-Spans":            {"false"},
		"X-Midt-Spans":                    {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	rdr := int64(534290)
	incSpan := false
	msg := Message{Type: CreateMessageType,
		Source:          "src",
		TransactionUUID: "test_transaction_id01",
		Headers:         []string{"key1", "value1", "key2", "value2", "key3", "value3"},
		Spans: [][]string{{"client1", "14678987563", "200"},
			{"client2", "146564565673", "500"}},
		Path: "/webpa-uuid",
	}
	msg.RequestDeliveryResponse = &rdr
	msg.IncludeSpans = &incSpan

	header, err := WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	assert.Equal(expectedHeader, header)

	// Missing MsgType and Source and Transaction uuid
	msg = Message{
		Headers: []string{"key1", "value1", "key2", "value2", "key3", "value3"},
		Spans: [][]string{{"client1", "14678987563", "200"},
			{"client2", "146564565673", "500"}},
		Path: "/webpa-uuid",
	}
	header, err = WRPToHeader(&msg)
	assert.Nil(header)
	assert.NotNil(err)
	assert.Equal(ErrInvalidMsgType, err)

	expectedHeader = http.Header{
		"X-Midt-Msg-Type":                 {"Create"},
		"X-Midt-Source":                   {"src"},
		"X-Midt-Path":                     {"/webpa-uuid"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2", "key3", "value3"},
		"X-Midt-Request-Delivery-Reponse": {"0"},
		"X-Midt-Include-Spans":            {"true"},
		"X-Midt-Spans":                    {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	rdr = int64(0)
	incSpan = true
	msg = Message{Type: CreateMessageType,
		Source:          "src",
		TransactionUUID: "test_transaction_id01",
		Headers:         []string{"key1", "value1", "key2", "value2", "key3", "value3"},
		Spans: [][]string{{"client1", "14678987563", "200"},
			{"client2", "146564565673", "500"}},
		Path: "/webpa-uuid",
	}
	msg.RequestDeliveryResponse = &rdr
	msg.IncludeSpans = &incSpan

	header, err = WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	assert.Equal(expectedHeader, header)
}

func TestWRPToHeader_Retrieve(t *testing.T) {
	assert := assert.New(t)

	// Success case
	expectedHeader := http.Header{
		"X-Midt-Msg-Type":                 {"Retrieve"},
		"X-Midt-Source":                   {"src"},
		"X-Midt-Path":                     {"/webpa-uuid"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2", "key3", "value3"},
		"X-Midt-Request-Delivery-Reponse": {"534290"},
		"X-Midt-Include-Spans":            {"true"},
		"X-Midt-Spans":                    {"client1", "14678987563", "200", "client2", "146564565673", "500"},
	}
	rdr := int64(534290)
	incSpan := true
	msg := Message{Type: RetrieveMessageType,
		Source:          "src",
		TransactionUUID: "test_transaction_id01",
		Headers:         []string{"key1", "value1", "key2", "value2", "key3", "value3"},
		Spans: [][]string{{"client1", "14678987563", "200"},
			{"client2", "146564565673", "500"}},
		Path: "/webpa-uuid",
	}
	msg.RequestDeliveryResponse = &rdr
	msg.IncludeSpans = &incSpan

	header, err := WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	assert.Equal(expectedHeader, header)
}

func TestWRPToHeader_Update(t *testing.T) {
	assert := assert.New(t)

	// Success case
	expectedHeader := http.Header{
		"X-Midt-Msg-Type":                 {"Update"},
		"X-Midt-Source":                   {"src"},
		"X-Midt-Path":                     {"/webpa-uuid"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2", "key3", "value3"},
		"X-Midt-Request-Delivery-Reponse": {"534290"},
		"X-Midt-Spans":                    {"client1", "14678987563"},
	}
	rdr := int64(534290)
	msg := Message{Type: UpdateMessageType,
		Source:          "src",
		TransactionUUID: "test_transaction_id01",
		Headers:         []string{"key1", "value1", "key2", "value2", "key3", "value3"},
		Spans:           [][]string{{"client1", "14678987563"}},
		Path:            "/webpa-uuid",
	}
	msg.RequestDeliveryResponse = &rdr

	header, err := WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	assert.Equal(expectedHeader, header)
}

func TestWRPToHeader_Delete(t *testing.T) {
	assert := assert.New(t)

	// Success case
	expectedHeader := http.Header{
		"X-Midt-Msg-Type":                 {"Delete"},
		"X-Midt-Source":                   {"src"},
		"X-Midt-Path":                     {"/webpa-uuid"},
		"X-Midt-Transaction-Uuid":         {"test_transaction_id01"},
		"X-Midt-Headers":                  {"key1", "value1", "key2", "value2", "key3", "value3"},
		"X-Midt-Request-Delivery-Reponse": {"534290"},
		"X-Midt-Include-Spans":            {"true"},
	}
	rdr := int64(534290)
	incSpan := true
	msg := Message{Type: DeleteMessageType,
		Source:          "src",
		TransactionUUID: "test_transaction_id01",
		Headers:         []string{"key1", "value1", "key2", "value2", "key3", "value3"},
		Path:            "/webpa-uuid",
	}
	msg.RequestDeliveryResponse = &rdr
	msg.IncludeSpans = &incSpan

	header, err := WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	assert.Equal(expectedHeader, header)

	// Basic
	msg = Message{Type: DeleteMessageType,
		Source: "src",
		Path:   "/webpa-uuid",
	}
	expectedHeader = http.Header{
		"X-Midt-Msg-Type": {"Delete"},
		"X-Midt-Source":   {"src"},
		"X-Midt-Path":     {"/webpa-uuid"},
	}
	header, err = WRPToHeader(&msg)
	assert.NotNil(header)
	assert.Nil(err)
	assert.Equal(expectedHeader, header)
}
