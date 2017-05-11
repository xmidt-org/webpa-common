package aws

import (
	"testing"
	"github.com/gorilla/mux"
	"time"
	"net/http/httptest"
	"net/http"
	"strings"
	"github.com/stretchr/testify/assert"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/mock"
	"net/url"
	"fmt"
	"io/ioutil"
	"encoding/json"
)

type ErrResp struct {
	Code int
	Message string
}

const (
	NOTIF_MSG = `{
  "Type" : "Notification",
  "MessageId" : "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
  "TopicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
  "Subject" : "My First Message",
  "Message" : "Hello world!",
  "Timestamp" : "2012-05-02T00:54:06.655Z",
  "SignatureVersion" : "1",
  "Signature" : "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
  "SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
  "UnsubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96",
  "MessageAttributes" : {
    "scytale.env" : {"Type":"String","Value":"test"}
  } }`
	TEST_NOTIF_MSG = `{
  "Type" : "Notification",
  "MessageId" : "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
  "TopicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
  "Subject" : "My First Message",
  "Message" : "Hello world!",
  "Timestamp" : "2012-05-02T00:54:06.655Z",
  "SignatureVersion" : "1",
  "Signature" : "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
  "SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
  "UnsubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96",
  "MessageAttributes" : {
    "scytale.env" : {"Type":"String","Value":"Invalid"}
  } }`
)

func SetUpTestSNSServer() (*SNSServer, *MockSVC, *mux.Router)  {
	
	v := SetUpTestViperInstance(TEST_AWS_CONFIG)
	
	awsCfg, _ := NewAWSConfig(v.Sub(AWSKey))
	m := &MockSVC{}
	
	ss := &SNSServer{
		Config: *awsCfg,
		SVC: m,
	}
	
	selfURL := &url.URL{
		Scheme:   "http",
		Host:     "127.0.0.1:8090",
	}
	
	r := mux.NewRouter()
	ss.Initialize(r, selfURL, nil, nil)
	
	return ss, m, r
}

func TestSubscribeSuccess(t *testing.T) {
	
	assert  := assert.New(t)
	expectedSubArn := "pending confirmation"
	
	ss, m, _ := SetUpTestSNSServer()
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
													SubscriptionArn: &expectedSubArn},nil)
	ss.PrepareAndStart()
	
	m.AssertExpectations(t)
	
	// wait such that listenSubscriptionData go routine will update the SubscriptionArn value
	time.Sleep(1*time.Second)
	
	assert.Equal(ss.subscriptionArn.Load().(string), expectedSubArn)	
}

func TestSubscribeError(t *testing.T) {
	
	assert  := assert.New(t)
	
	ss, m, _ := SetUpTestSNSServer()
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{},
		fmt.Errorf("%s", "InvalidClientTokenId"))
	
	ss.PrepareAndStart()
	
	m.AssertExpectations(t)
	
	// wait such that listenSubscriptionData will update the SubscriptionArn value
	time.Sleep(1*time.Second)
	
	assert.Nil(ss.subscriptionArn.Load())
	
}

func TestUnsubscribeSuccess (t *testing.T) {
	
	ss, m, _ := SetUpTestSNSServer()
	testSubscribe(t,m,ss)
	
	m.On("Unsubscribe",mock.AnythingOfType("*sns.UnsubscribeInput")).Return(&sns.UnsubscribeOutput{},nil)
	
	// wait such that listenSubscriptionData go routine will update the SubscriptionArn value
	time.Sleep(1*time.Second)
	
	ss.Unsubscribe()
	
	m.AssertExpectations(t)
}

func TestSetSNSRoutes_SubConf(t *testing.T) {
	
	assert  := assert.New(t)
	expectedSubArn := "pending confirmation"
	confSubArn := "testRoute"
	ss, m, r := SetUpTestSNSServer()
	
	ts := httptest.NewServer(r)
	
	subConfUrl := fmt.Sprintf("%s%s", ts.URL,ss.Config.Sns.UrlPath)
	
	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type","SubscriptionConfirmation")
	
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
													SubscriptionArn: &expectedSubArn},nil)
	ss.PrepareAndStart()
	
	// wait such that listenSubscriptionData go routine will update the SubscriptionArn value
	//time.Sleep(1*time.Second)
	
	// mocking SNS ConfirmSubscription response
	m.On("ConfirmSubscription",mock.AnythingOfType("*sns.ConfirmSubscriptionInput")).Return(&sns.ConfirmSubscriptionOutput{
													SubscriptionArn: &confSubArn},nil)
	
	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	
	m.AssertExpectations(t)
	
	assert.Equal(res.StatusCode,http.StatusOK)
	
}

func TestNotificationHandleSuccess(t *testing.T) {
	assert  := assert.New(t)
	pub_msg := "Hello world!"
	ss, m, _ := SetUpTestSNSServer()
	
	testSubscribe(t,m,ss)
	testSubConf(t,m,ss)
	
	// mocking SNS Publish response
	m.On("Publish",mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{},nil)
	
	ss.PublishMessage(pub_msg)
	
	time.Sleep(1*time.Second)
	m.AssertExpectations(t)
	
	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ss.SelfUrl.String() + ss.Config.Sns.UrlPath, strings.NewReader(NOTIF_MSG))
	req.Header.Add("x-amz-sns-message-type","Notification")
	req.Header.Add("x-amz-sns-subscription-arn","testSubscriptionArn")
	
	w := httptest.NewRecorder()
	message := ss.NotificationHandle(w, req)
	resp := w.Result()
    
    assert.Equal(http.StatusOK, resp.StatusCode)
    assert.Equal(message, []byte(pub_msg))
}


func TestNotificationHandleError_SubArnMismatch(t *testing.T) {
	assert  := assert.New(t)
	ss, m, _ := SetUpTestSNSServer()
	
	testSubscribe(t,m,ss)
	testSubConf(t,m,ss)
	testPublish(t,m,ss)
	
	m.AssertExpectations(t)
	
	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ss.SelfUrl.String() + ss.Config.Sns.UrlPath, strings.NewReader(NOTIF_MSG))
	req.Header.Add("x-amz-sns-message-type","Notification")
	req.Header.Add("x-amz-sns-subscription-arn","Invalid")
	
	w := httptest.NewRecorder()
	message := ss.NotificationHandle(w, req)
	resp := w.Result()
	
	errMsg := new(ErrResp)
	errResp, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(errResp), errMsg)
    
    assert.Equal(http.StatusBadRequest, resp.StatusCode)
    assert.Nil(message)
    assert.Equal(errMsg.Code,http.StatusBadRequest)
    assert.Equal(errMsg.Message,"SubscriptionARN does not match")
}

func TestNotificationHandleError_ReadErr(t *testing.T) {
	assert  := assert.New(t)
	ss, m, _ := SetUpTestSNSServer()
	
	testSubscribe(t,m,ss)
	testSubConf(t,m,ss)
	testPublish(t,m,ss)
	
	m.AssertExpectations(t)
	
	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ss.SelfUrl.String() + ss.Config.Sns.UrlPath, 
		strings.NewReader(TEST_SNS_ERR_MSG))
	req.Header.Add("x-amz-sns-message-type","Notification")
	req.Header.Add("x-amz-sns-subscription-arn","testSubscriptionArn")
	
	w := httptest.NewRecorder()
	message := ss.NotificationHandle(w, req)
	resp := w.Result()
	
	errMsg := new(ErrResp)
	errResp, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(errResp), errMsg)
    
    assert.Equal(http.StatusBadRequest, resp.StatusCode)
    assert.Nil(message)
    assert.Equal(errMsg.Code,http.StatusBadRequest)
    assert.Equal(errMsg.Message,"request body error")
}

func TestNotificationHandleError_MsgEnvMismatch(t *testing.T) {
	assert  := assert.New(t)
	ss, m, _ := SetUpTestSNSServer()
	
	testSubscribe(t,m,ss)
	testSubConf(t,m,ss)
	testPublish(t,m,ss)
	
	m.AssertExpectations(t)
	
	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ss.SelfUrl.String() + ss.Config.Sns.UrlPath, 
		strings.NewReader(TEST_NOTIF_MSG))
	req.Header.Add("x-amz-sns-message-type","Notification")
	req.Header.Add("x-amz-sns-subscription-arn","testSubscriptionArn")
	
	w := httptest.NewRecorder()
	message := ss.NotificationHandle(w, req)
	resp := w.Result()
	
	errMsg := new(ErrResp)
	errResp, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(errResp), errMsg)
    
    assert.Equal(http.StatusBadRequest, resp.StatusCode)
    assert.Nil(message)
    assert.Equal(errMsg.Code,http.StatusBadRequest)
    assert.Equal(errMsg.Message,"SNS Msg config env does not match")
}
