package aws

import (
	"testing"
	"net/http/httptest"
	"strings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_SNS_MSG = `{
  "Type" : "Notification",
  "MessageId" : "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
  "TopicArn" : "arn:aws:sns:us-west-2:123456789012:MyTopic",
  "Subject" : "My First Message",
  "Message" : "Hello world!",
  "Timestamp" : "2012-05-02T00:54:06.655Z",
  "SignatureVersion" : "1",
  "Signature" : "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
  "SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
  "UnsubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96"
  }`
)

func TestSuccessDecodeJSONMessage(t *testing.T) {
	
	req := httptest.NewRequest("POST", "/foo", strings.NewReader(TEST_SNS_MSG))
	msg := new(SNSMessage)
	assert  := assert.New(t)
	require := require.New(t)
	
	payload, err := DecodeJSONMessage(req, msg)
	require.NotNil(payload)
	assert.Nil(err)
	assert.Equal([]byte(TEST_SNS_MSG), payload)
	assert.Equal("Notification", msg.Type)
	assert.Equal("Hello world!",msg.Message)
	assert.Equal("My First Message", msg.Subject)
	assert.Equal(
		"https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96", 
		msg.UnsubscribeURL)
	assert.Len(payload, len([]byte(TEST_SNS_MSG)))
}

func TestErrDecodeJSONMessage(t *testing.T) {
	
	snsErrTypeMsg := `{
  "Type" : "Notification",
  "MessageId" : 22324,
  "TopicArn" : "arn:aws:sns:us-west-2:123456789012:MyTopic",
  "Subject" : "My First Message",
  "Message" : "Hello world!",
  "Timestamp" : "2012-05-02T00:54:06.655Z",
  "SignatureVersion" : "1",
  "Signature" : "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
  "SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
  "UnsubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96"
  }`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(snsErrTypeMsg))
	msg := new(SNSMessage)
	assert  := assert.New(t)
	
	payload, err := DecodeJSONMessage(req, msg)
	assert.Nil(payload)
	assert.NotNil(err)
}

func TestEmptyDecodeJSONMessage(t *testing.T) {
	
	snsErrTypeMsg := ``
	req := httptest.NewRequest("POST", "/test", strings.NewReader(snsErrTypeMsg))
	msg := new(SNSMessage)
	assert  := assert.New(t)
	
	payload, err := DecodeJSONMessage(req, msg)
	assert.Nil(payload)
	assert.NotNil(err)
	assert.Equal(ErrJsonEmpty,err)
}
