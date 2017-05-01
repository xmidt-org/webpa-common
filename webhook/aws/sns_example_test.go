package aws

import (
	"net/http/httptest"
	"net/http"
	"strings"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/aws/aws-sdk-go/service/sns"
)

const (
	TEST_SUB_MSG = `{
  "Type" : "SubscriptionConfirmation",
  "MessageId" : "165545c9-2a5c-472c-8df2-7ff2be2b3b1b",
  "Token" : "2336412f37fb687f5d51e6e241d09c805a5a57b30d712f794cc5f6a988666d92768dd60a747ba6f3beb71854e285d6ad02428b09ceece29417f1f02d609c582afbacc99c583a916b9981dd2728f4ae6fdb82efd087cc3b7849e05798d2d2785c03b0879594eeac82c01f235d0e717736",
  "TopicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
  "Message" : "You have chosen to subscribe to the topic arn:aws:sns:us-east-1:1234:test-topic.\nTo confirm the subscription, visit the SubscribeURL included in this message.",
  "SubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn:aws:sns:us-west-2:123456789012:MyTopic&Token=2336412f37fb687f5d51e6e241d09c805a5a57b30d712f794cc5f6a988666d92768dd60a747ba6f3beb71854e285d6ad02428b09ceece29417f1f02d609c582afbacc99c583a916b9981dd2728f4ae6fdb82efd087cc3b7849e05798d2d2785c03b0879594eeac82c01f235d0e717736",
  "Timestamp" : "2012-04-26T20:45:04.751Z",
  "SignatureVersion" : "1",
  "Signature" : "EXAMPLEpH+DcEwjAPg8O9mY8dReBSwksfg2S7WKQcikcNKWLQjwu6A4VbeS0QHVCkhRS7fUQvi2egU3N858fiTDN6bkkOxYDVrY0Ad8L10Hs3zH81mtnPk5uvvolIC1CXGu43obcgFxeL3khZl8IKvO61GWB6jI9b5+gLPoBc1Q=",
  "SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem"
  }`
	TEST_HOOK = `{
		"config": {
			"url": "http://127.0.0.1:8080/test",
			"content_type": "json",
			"secret": ""
		},
		"matcher": {
			"device_id": [
				".*"
			]
		},
		"events": [
			"transaction-status",
			"SYNC_NOTIFICATION"
		]
	}`
)

func TestSNSReadyAndPublishSuccess(t *testing.T) {
	
	assert  := assert.New(t)
	expectedSubArn := "pending confirmation"
	confSubArn := "testSubscriptionArn"
	
	ss, m, _ := SetUpTestSNSServer()
	
	// mocking SNS subscribe response
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
													SubscriptionArn: &expectedSubArn},nil)
	ss.PrepareAndStart()
	
	// mocking SNS ConfirmSubscription response
	m.On("ConfirmSubscription",mock.AnythingOfType("*sns.ConfirmSubscriptionInput")).Return(&sns.ConfirmSubscriptionOutput{
													SubscriptionArn: &confSubArn},nil)

	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", ss.SelfUrl.String() + ss.Config.Sns.UrlPath, strings.NewReader(TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type","SubscriptionConfirmation")
	
	w := httptest.NewRecorder()
	ss.SubscribeConfirmHandle(w, req)
	resp := w.Result()
    
    assert.Equal(http.StatusOK, resp.StatusCode)	
    
	// wait such that listenSubscriptionData go routine will update the SubscriptionArn value
	time.Sleep(1*time.Second)

	assert.Equal(ss.subscriptionArn.Load().(string), confSubArn)
	
	// mocking SNS Publish response
	m.On("Publish",mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{},nil)
	
	ss.PublishMessage(TEST_HOOK)
	
	// wait such that listenAndPublishMessage go routine will publish message
	time.Sleep(1*time.Second)
	
	m.AssertExpectations(t)
}


