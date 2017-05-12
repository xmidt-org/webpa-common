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

func TestSNSReadyAndPublishSuccess(t *testing.T) {
	
	ss, m, _ := SetUpTestSNSServer()
	
	testSubscribe(t,m,ss)
	testSubConf(t,m,ss)
	testPublish(t,m,ss)
	
	m.AssertExpectations(t)
}

func TestSNSReadyToNotReadySwitchAndBack(t *testing.T) {
	assert  := assert.New(t)
	expectedSubArn := "pending confirmation"
	
	ss, m, _ := SetUpTestSNSServer()
	
	testSubscribe(t,m,ss)
	testSubConf(t,m,ss)
	testPublish(t,m,ss)
	
	// mocking SNS subscribe response
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
													SubscriptionArn: &expectedSubArn},nil)
	// Subscribe again to change SNS to not ready
	ss.Subscribe()
	
	time.Sleep(1*time.Second)
	
	assert.Equal(ss.subscriptionArn.Load().(string), expectedSubArn)
	
	// listenAndPublishMessage is terminated hence no mock need for PublishInput
	ss.PublishMessage(TEST_HOOK)
	
	// SNS Ready and Publish again
	testSubConf(t,m,ss)
	testPublish(t,m,ss)
	
	m.AssertExpectations(t)
}

func testSubscribe(t *testing.T, m *MockSVC, ss *SNSServer) {
	assert  := assert.New(t)
	expectedSubArn := "pending confirmation"
	
	// mocking SNS subscribe response
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
													SubscriptionArn: &expectedSubArn},nil)
	ss.PrepareAndStart()
	
	// wait such that listenSubscriptionData go routine will update the SubscriptionArn value
	time.Sleep(1*time.Second)
	
	assert.Equal(ss.subscriptionArn.Load().(string), expectedSubArn)
}

func testSubConf(t *testing.T, m *MockSVC, ss *SNSServer) {
	assert  := assert.New(t)
	
	confSubArn := "testSubscriptionArn"
	
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
}

func testPublish(t *testing.T, m *MockSVC, ss *SNSServer) {
	// mocking SNS Publish response
	m.On("Publish",mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{},nil)
	
	ss.PublishMessage(TEST_HOOK)
	
	// wait such that listenAndPublishMessage go routine will publish message
	time.Sleep(1*time.Second)
}

