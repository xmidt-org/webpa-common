package wrp

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"net/http"
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
	expectedMsgType := []MessageType {
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
		assert.Equal(msgType,expectedMsgType[i])
	}
}

func TestHeaderToWRP_Auth(t *testing.T) {
	assert := assert.New(t)
	
	// Success case
	testHeader := http.Header{"X-Midt-Msg-Type" : {"Auth"},"X-Midt-Status" : {"200"}}
	msg,err := HeaderToWRP(testHeader)
	assert.NotNil(msg)
	assert.Nil(err)
	assert.Equal(msg.MessageType(),AuthMessageType)	
	assert.Equal(*msg.Status,int64(200))
	
	// Invalid status
	testHeader = http.Header{"X-Midt-Msg-Type" : {"Auth"},"X-Midt-Status" : {"Invalid"}}
	msg,err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	
	// Invalid MsgType
	testHeader = http.Header{"X-Midt-Msg-Type" : {"Invalid"},"X-Midt-Status" : {"400"}}
	msg,err = HeaderToWRP(testHeader)
	assert.Nil(msg)
	assert.NotNil(err)
	assert.Equal("Invalid Message Type header string", err.Error())
}