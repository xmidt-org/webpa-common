package aws

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSNSReadyAndPublishSuccess(t *testing.T) {

	ss, m, mv, _ := SetUpTestSNSServer(t)

	testSubscribe(t, m, ss)
	testSubConf(t, m, mv, ss)
	testPublish(t, m, ss)

	m.AssertExpectations(t)
}

func TestSNSReadyToNotReadySwitchAndBack(t *testing.T) {
	expectedSubArn := "pending confirmation"

	ss, m, mv, _ := SetUpTestSNSServer(t)

	testSubscribe(t, m, ss)
	testSubConf(t, m, mv, ss)
	testPublish(t, m, ss)

	// mocking SNS subscribe response
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)
	// Subscribe again will not change the ready state
	// as the subArn value stored locally and in AWS are still the same
	ss.Subscribe()

	assert.Equal(t, ss.subscriptionArn.Load().(string), "testSubscriptionArn")

	// Test Publish
	testPublish(t, m, ss)

	// SNS Ready and Publish again
	testSubConf(t, m, mv, ss)
	testPublish(t, m, ss)

	m.AssertExpectations(t)
}

func testSubscribe(t *testing.T, m *MockSVC, ss *SNSServer) {
	expectedSubArn := "pending confirmation"

	// mocking SNS subscribe response
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)
	ss.PrepareAndStart()
	assert.Nil(t, ss.subscriptionArn.Load())
}

func testSubConf(t *testing.T, m *MockSVC, mv *MockValidator, ss *SNSServer) {
	assert := assert.New(t)

	confSubArn := "testSubscriptionArn"

	// mocking SNS ConfirmSubscription response
	m.On("ConfirmSubscription", mock.AnythingOfType("*sns.ConfirmSubscriptionInput")).Return(&sns.ConfirmSubscriptionOutput{
		SubscriptionArn: &confSubArn}, nil)
	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil).Once()

	// mocking SNS ListSubscriptionsByTopic response to empty list
	m.On("ListSubscriptionsByTopic", mock.AnythingOfType("*sns.ListSubscriptionsByTopicInput")).Return(
		&sns.ListSubscriptionsByTopicOutput{Subscriptions: []*sns.Subscription{}}, nil)

	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", ss.SelfUrl.String()+ss.Config.Sns.UrlPath, strings.NewReader(TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type", "SubscriptionConfirmation")

	w := httptest.NewRecorder()
	ss.SubscribeConfirmHandle(w, req)
	resp := w.Result()

	assert.Equal(http.StatusOK, resp.StatusCode)
	time.Sleep(1 * time.Second)
	assert.Equal(ss.subscriptionArn.Load().(string), confSubArn)

}

func testPublish(t *testing.T, m *MockSVC, ss *SNSServer) {
	// mocking SNS Publish response
	m.On("Publish", mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{}, nil)

	err := ss.PublishMessage(TEST_HOOK)
	if nil != err {
		panic(err)
	}

	// wait such that listenAndPublishMessage go routine will publish message
	time.Sleep(1 * time.Second)

}

func TestSNSSubConfValidateErr(t *testing.T) {
	assert := assert.New(t)

	ss, m, mv, _ := SetUpTestSNSServer(t)

	testSubscribe(t, m, ss)

	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(false,
		fmt.Errorf("%s", SNS_VALIDATION_ERR))

	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", ss.SelfUrl.String()+ss.Config.Sns.UrlPath, strings.NewReader(TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type", "SubscriptionConfirmation")

	w := httptest.NewRecorder()
	ss.SubscribeConfirmHandle(w, req)
	resp := w.Result()

	assert.Equal(http.StatusBadRequest, resp.StatusCode)
	errMsg := new(ErrResp)
	errResp, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(errMsg.Code, http.StatusBadRequest)
	assert.Equal(errMsg.Message, SNS_VALIDATION_ERR)

	m.AssertExpectations(t)
	mv.AssertExpectations(t)
}

func TestSNSReadyUnsubscribeOldSubscriptions(t *testing.T) {
	assert := assert.New(t)
	ss, m, mv, _ := SetUpTestSNSServer(t)

	testSubscribe(t, m, ss)

	confSubArn := "testSubscriptionArn"

	// mocking SNS ConfirmSubscription response
	m.On("ConfirmSubscription", mock.AnythingOfType("*sns.ConfirmSubscriptionInput")).Return(&sns.ConfirmSubscriptionOutput{
		SubscriptionArn: &confSubArn}, nil)
	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil).Once()

	// mocking SNS ListSubscriptionsByTopic response to list
	sub1 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/api/v2/aws/sns/1503357402"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test1"),
	}
	sub2 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/api/v2/aws/sns"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test2"),
	}
	m.On("ListSubscriptionsByTopic", mock.AnythingOfType("*sns.ListSubscriptionsByTopicInput")).Return(
		&sns.ListSubscriptionsByTopicOutput{Subscriptions: []*sns.Subscription{sub1, sub2}}, nil)

	// mocking Unsubscribe call
	m.On("Unsubscribe", &sns.UnsubscribeInput{SubscriptionArn: aws.String("test2")}).Return(&sns.UnsubscribeOutput{}, nil)

	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", ss.SelfUrl.String()+ss.Config.Sns.UrlPath, strings.NewReader(TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type", "SubscriptionConfirmation")

	w := httptest.NewRecorder()
	ss.SubscribeConfirmHandle(w, req)
	resp := w.Result()

	assert.Equal(http.StatusOK, resp.StatusCode)

	// wait such that listenSubscriptionData go routine will update the SubscriptionArn value
	time.Sleep(1 * time.Second)

	assert.Equal(ss.subscriptionArn.Load().(string), confSubArn)

	m.AssertExpectations(t)
	mv.AssertExpectations(t)
}
