package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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
	TEST_UNIX_TIME = 1503357402
)

func testNow() time.Time {
	return time.Unix(TEST_UNIX_TIME, 0)
}

func SetUpTestViperInstance(config string) *viper.Viper {

	cfg := bytes.NewBufferString(config)
	v := viper.New()
	v.SetConfigType("json")
	v.ReadConfig(cfg)
	return v
}

func SetUpTestSNSServer() (*SNSServer, *MockSVC, *MockValidator, *mux.Router) {

	v := SetUpTestViperInstance(TEST_AWS_CONFIG)

	awsCfg, _ := NewAWSConfig(v)
	m := &MockSVC{}
	mv := &MockValidator{}

	ss := &SNSServer{
		Config:       *awsCfg,
		SVC:          m,
		SNSValidator: mv,
	}

	r := mux.NewRouter()
	ss.Initialize(r, nil, nil, nil, testNow)

	return ss, m, mv, r
}

func TestSubscribeSuccess(t *testing.T) {
	fmt.Println("\n\nTestSubscribeSuccess")

	assert := assert.New(t)
	expectedSubArn := "pending confirmation"

	ss, m, _, _ := SetUpTestSNSServer()
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)
	ss.PrepareAndStart()

	m.AssertExpectations(t)

	// wait such that listenSubscriptionData go routine will update the SubscriptionArn value
	time.Sleep(1 * time.Second)

	assert.Equal(ss.subscriptionArn.Load().(string), expectedSubArn)
}

func TestSubscribeError(t *testing.T) {
	fmt.Println("\n\nTestSubscribeError")

	assert := assert.New(t)

	ss, m, _, _ := SetUpTestSNSServer()
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{},
		fmt.Errorf("%s", "InvalidClientTokenId"))

	ss.PrepareAndStart()

	m.AssertExpectations(t)

	// wait such that listenSubscriptionData will update the SubscriptionArn value
	time.Sleep(1 * time.Second)

	assert.Nil(ss.subscriptionArn.Load())

}

func TestUnsubscribeSuccess(t *testing.T) {
	fmt.Println("\n\nTestUnsubscribeSuccess")

	ss, m, mv, _ := SetUpTestSNSServer()
	testSubscribe(t, m, ss)
	testSubConf(t, m, mv, ss)

	expectedInput := &sns.UnsubscribeInput{
		SubscriptionArn: aws.String("testSubscriptionArn"), // Required
	}
	m.On("Unsubscribe", expectedInput).Return(&sns.UnsubscribeOutput{}, nil)

	// wait such that listenSubscriptionData go routine will update the SubscriptionArn value
	time.Sleep(1 * time.Second)

	ss.Unsubscribe("")

	m.AssertExpectations(t)
}

func TestUnsubscribeWithSubArn(t *testing.T) {
	fmt.Println("\n\nTestUnsubscribeSuccess")

	ss, m, _, _ := SetUpTestSNSServer()
	testSubscribe(t, m, ss)

	expectedInput := &sns.UnsubscribeInput{
		SubscriptionArn: aws.String("subArn"), // Required
	}
	m.On("Unsubscribe", expectedInput).Return(&sns.UnsubscribeOutput{}, nil)

	// wait such that listenSubscriptionData go routine will update the SubscriptionArn value
	time.Sleep(1 * time.Second)

	ss.Unsubscribe("subArn")

	m.AssertExpectations(t)
}

func TestSetSNSRoutes_SubConf(t *testing.T) {
	fmt.Println("\n\nTestSetSNSRoutes_SubConf")

	assert := assert.New(t)
	expectedSubArn := "pending confirmation"
	confSubArn := "testRoute"
	ss, m, mv, r := SetUpTestSNSServer()

	ts := httptest.NewServer(r)

	subConfUrl := fmt.Sprintf("%s%s/%d", ts.URL, ss.Config.Sns.UrlPath, TEST_UNIX_TIME)

	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type", "SubscriptionConfirmation")

	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)
	ss.PrepareAndStart()

	// mocking SNS ConfirmSubscription response
	m.On("ConfirmSubscription", mock.AnythingOfType("*sns.ConfirmSubscriptionInput")).Return(&sns.ConfirmSubscriptionOutput{
		SubscriptionArn: &confSubArn}, nil)

	// mocking SNS ListSubscriptionsByTopic response to empty list
	m.On("ListSubscriptionsByTopic", mock.AnythingOfType("*sns.ListSubscriptionsByTopicInput")).Return(
		&sns.ListSubscriptionsByTopicOutput{Subscriptions: []*sns.Subscription{}}, nil)

	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil)

	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	m.AssertExpectations(t)

	mv.AssertExpectations(t)

	assert.Equal(res.StatusCode, http.StatusOK)

}

func TestNotificationHandleSuccess(t *testing.T) {
	fmt.Println("\n\nTestNotificationHandleSuccess")

	assert := assert.New(t)
	pub_msg := "Hello world!"
	ss, m, mv, _ := SetUpTestSNSServer()

	testSubscribe(t, m, ss)
	testSubConf(t, m, mv, ss)

	// mocking SNS Publish response
	m.On("Publish", mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{}, nil)

	ss.PublishMessage(pub_msg)

	time.Sleep(1 * time.Second)

	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil)

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ss.SelfUrl.String()+ss.Config.Sns.UrlPath, strings.NewReader(NOTIF_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	w := httptest.NewRecorder()
	message := ss.NotificationHandle(w, req)
	resp := w.Result()

	assert.Equal(http.StatusOK, resp.StatusCode)
	assert.Equal(message, []byte(pub_msg))

	m.AssertExpectations(t)

	mv.AssertExpectations(t)
}

func TestNotificationHandleError_SubArnMismatch(t *testing.T) {
	fmt.Println("\n\nTestNotificationHandleError_SubArnMismatch")

	assert := assert.New(t)
	ss, m, mv, _ := SetUpTestSNSServer()

	testSubscribe(t, m, ss)
	testSubConf(t, m, mv, ss)
	testPublish(t, m, ss)

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ss.SelfUrl.String()+ss.Config.Sns.UrlPath, strings.NewReader(NOTIF_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "Invalid")

	w := httptest.NewRecorder()
	message := ss.NotificationHandle(w, req)
	resp := w.Result()

	errMsg := new(ErrResp)
	errResp, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusInternalServerError, resp.StatusCode)
	assert.Nil(message)
	assert.Equal(errMsg.Code, http.StatusInternalServerError)
	assert.Equal(errMsg.Message, "SubscriptionARN does not match")

	m.AssertExpectations(t)
}

func TestNotificationHandleError_ReadErr(t *testing.T) {
	fmt.Println("\n\nTestNotificationHandleError_ReadErr")

	assert := assert.New(t)
	ss, m, mv, _ := SetUpTestSNSServer()

	testSubscribe(t, m, ss)
	testSubConf(t, m, mv, ss)
	testPublish(t, m, ss)

	m.AssertExpectations(t)

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ss.SelfUrl.String()+ss.Config.Sns.UrlPath,
		strings.NewReader(TEST_SNS_ERR_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	w := httptest.NewRecorder()
	message := ss.NotificationHandle(w, req)
	resp := w.Result()

	errMsg := new(ErrResp)
	errResp, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusBadRequest, resp.StatusCode)
	assert.Nil(message)
	assert.Equal(errMsg.Code, http.StatusBadRequest)
	assert.Equal(errMsg.Message, "request body error")
}

func TestNotificationHandleError_MsgEnvMismatch(t *testing.T) {
	fmt.Println("\n\nTestNotificationHandleError_MsgEnvMismatch")

	assert := assert.New(t)
	ss, m, mv, _ := SetUpTestSNSServer()

	testSubscribe(t, m, ss)
	testSubConf(t, m, mv, ss)
	testPublish(t, m, ss)

	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil)

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ss.SelfUrl.String()+ss.Config.Sns.UrlPath,
		strings.NewReader(TEST_NOTIF_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	w := httptest.NewRecorder()
	message := ss.NotificationHandle(w, req)
	resp := w.Result()

	errMsg := new(ErrResp)
	errResp, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusBadRequest, resp.StatusCode)
	assert.Nil(message)
	assert.Equal(errMsg.Code, http.StatusBadRequest)
	assert.Equal(errMsg.Message, "SNS Msg config env does not match")

	m.AssertExpectations(t)
	mv.AssertExpectations(t)
}

func TestNotificationHandleError_ValidationErr(t *testing.T) {
	fmt.Println("\n\nTestNotificationHandleError_ValidationErr")

	assert := assert.New(t)
	ss, m, mv, _ := SetUpTestSNSServer()

	testSubscribe(t, m, ss)
	testSubConf(t, m, mv, ss)
	testPublish(t, m, ss)

	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(false,
		fmt.Errorf("%s", SNS_VALIDATION_ERR))

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ss.SelfUrl.String()+ss.Config.Sns.UrlPath,
		strings.NewReader(NOTIF_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	w := httptest.NewRecorder()
	message := ss.NotificationHandle(w, req)
	resp := w.Result()

	assert.Equal(http.StatusBadRequest, resp.StatusCode)

	errMsg := new(ErrResp)
	errResp, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Nil(message)
	assert.Equal(errMsg.Code, http.StatusBadRequest)
	assert.Equal(errMsg.Message, SNS_VALIDATION_ERR)

	m.AssertExpectations(t)
	mv.AssertExpectations(t)
}
