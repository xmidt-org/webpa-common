package wrp

import (
	"testing"
	"github.com/stretchr/testify/assert"
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