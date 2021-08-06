package aws

import (
	"bytes"
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
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/logging"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

const (
	TEST_SNS_CFG = `{
	"aws": {
        "accessKey": "test-accessKey",
        "secretKey": "test-secretKey",
        "env": "test",
        "sns" : {
	        "region" : "us-east-1",
            "protocol" : "http",
			"topicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
			"urlPath" : "/sns/"
    } } }`
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

func SetUpTestSNSServer(t *testing.T) (*SNSServer, *MockSVC, *MockValidator, *mux.Router) {
	return SetUpTestSNSServerWithChannelSize(t, 50)
}

func SetUpTestSNSServerWithChannelSize(t *testing.T, channelSize int64) (*SNSServer, *MockSVC, *MockValidator, *mux.Router) {

	v := SetUpTestViperInstance(TEST_AWS_CONFIG)

	awsCfg, _ := NewAWSConfig(v)
	m := &MockSVC{}
	mv := &MockValidator{}

	ss := &SNSServer{
		Config:               *awsCfg,
		SVC:                  m,
		SNSValidator:         mv,
		channelSize:          channelSize,
		channelClientTimeout: 30 * time.Second,
	}

	r := mux.NewRouter()
	logger := logging.NewTestLogger(nil, t)
	registry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics)
	awsMetrics := ApplyMetricsData(registry)
	ss.Initialize(r, nil, "", nil, logger, awsMetrics, testNow)

	return ss, m, mv, r
}

func TestSubscribeSuccess(t *testing.T) {
	fmt.Println("\n\nTestSubscribeSuccess")

	expectedSubArn := "pending confirmation"

	ss, m, _, _ := SetUpTestSNSServer(t)
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)
	ss.PrepareAndStart()

	m.AssertExpectations(t)

	assert.Nil(t, ss.subscriptionArn.Load())
}

func TestSubscribeError(t *testing.T) {
	fmt.Println("\n\nTestSubscribeError")

	assert := assert.New(t)

	ss, m, _, _ := SetUpTestSNSServer(t)
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{},
		fmt.Errorf("%s", "InvalidClientTokenId"))

	ss.PrepareAndStart()

	m.AssertExpectations(t)
	assert.Nil(ss.subscriptionArn.Load())
}

func TestUnsubscribeSuccess(t *testing.T) {
	fmt.Println("\n\nTestUnsubscribeSuccess")

	ss, m, mv, _ := SetUpTestSNSServer(t)
	testSubscribe(t, m, ss)
	testSubConf(t, m, mv, ss)

	expectedInput := &sns.UnsubscribeInput{
		SubscriptionArn: aws.String("testSubscriptionArn"), // Required
	}
	m.On("Unsubscribe", expectedInput).Return(&sns.UnsubscribeOutput{}, nil)

	ss.Unsubscribe("")

	m.AssertExpectations(t)
}

func TestUnsubscribeWithSubArn(t *testing.T) {
	fmt.Println("\n\nTestUnsubscribeSuccess")

	ss, m, _, _ := SetUpTestSNSServer(t)
	testSubscribe(t, m, ss)

	expectedInput := &sns.UnsubscribeInput{
		SubscriptionArn: aws.String("subArn"), // Required
	}
	m.On("Unsubscribe", expectedInput).Return(&sns.UnsubscribeOutput{}, nil)

	ss.Unsubscribe("subArn")

	m.AssertExpectations(t)
}

func TestSetSNSRoutes_SubConf(t *testing.T) {
	fmt.Println("\n\nTestSetSNSRoutes_SubConf")

	assert := assert.New(t)
	expectedSubArn := "pending confirmation"
	confSubArn := "testRoute"
	ss, m, mv, r := SetUpTestSNSServer(t)

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

	time.Sleep(1 * time.Second)

	m.AssertExpectations(t)

	mv.AssertExpectations(t)

	assert.Equal(res.StatusCode, http.StatusOK)

}

func TestNotificationHandleSuccess(t *testing.T) {
	fmt.Println("\n\nTestNotificationHandleSuccess")

	assert := assert.New(t)
	pub_msg := "Hello world!"
	ss, m, mv, _ := SetUpTestSNSServer(t)

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
	ss, m, mv, _ := SetUpTestSNSServer(t)

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
	ss, m, mv, _ := SetUpTestSNSServer(t)

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
	ss, m, mv, _ := SetUpTestSNSServer(t)

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
	ss, m, mv, _ := SetUpTestSNSServer(t)

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

func TestPublishClientTimeout(t *testing.T) {
	fmt.Println("\n\nTestNotificationHandleError_ValidationErr")

	assert := assert.New(t)
	ss, m, _, _ := SetUpTestSNSServerWithChannelSize(t, 1)
	ss.channelClientTimeout = 1

	assert.NotPanics(func() { testPublish(t, m, ss) })
	assert.Panics(func() { testPublish(t, m, ss) })
}

func TestListSubscriptionsByMatchingEndpointSuccess(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ss, m, _, _ := SetUpTestSNSServer(t)

	sub1 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/api/v2/aws/sns/1503357402"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test1"),
	}
	sub2 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/api/v2/aws/sns/1444357402"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test2"),
	}
	sub3 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/api/v2/aws/sns/1412357402"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test3"),
	}
	sub4 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/api/v2/aws/sns/1563357402"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test4"),
	}
	sub5 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/api/v2/aws/sns"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test5"),
	}

	// mocking SNS ListSubscriptionsByTopic response to empty list
	m.On("ListSubscriptionsByTopic", mock.AnythingOfType("*sns.ListSubscriptionsByTopicInput")).Return(
		&sns.ListSubscriptionsByTopicOutput{Subscriptions: []*sns.Subscription{sub1, sub2, sub3, sub4, sub5}}, nil)

	unsubList, err := ss.ListSubscriptionsByMatchingEndpoint()

	assert.Nil(err)
	require.NotNil(unsubList)
	assert.Equal(3, unsubList.Len())
	item := unsubList.Front()
	assert.Equal("test2", item.Value.(string))
	item = item.Next()
	assert.Equal("test3", item.Value.(string))
	item = item.Next()
	assert.Equal("test5", item.Value.(string))

	m.AssertExpectations(t)
}

func TestListSubscriptionsByMatchingEndpointSuccessWithNextToken(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	v := SetUpTestViperInstance(TEST_SNS_CFG)

	awsCfg, _ := NewAWSConfig(v)
	m := &MockSVC{}

	ss := &SNSServer{
		Config:               *awsCfg,
		SVC:                  m,
		channelSize:          50,
		channelClientTimeout: 30 * time.Second,
	}

	logger := logging.NewTestLogger(nil, t)
	registry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics)
	awsMetrics := ApplyMetricsData(registry)
	ss.Initialize(nil, nil, "", nil, logger, awsMetrics, testNow)

	sub1 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/sns/1503357402"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test1"),
	}
	sub2 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/sns/1444357402"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test2"),
	}
	sub3 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/sns/1412357402"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test3"),
	}
	sub4 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/sns/1563357402"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test4"),
	}
	sub5 := &sns.Subscription{
		Endpoint:        aws.String("http://host:10000/sns/"),
		TopicArn:        aws.String("arn:aws:sns:us-east-1:1234:test-topic"),
		SubscriptionArn: aws.String("test5"),
	}

	// mocking SNS ListSubscriptionsByTopic response to empty list
	expectedFirstInput := &sns.ListSubscriptionsByTopicInput{
		TopicArn: aws.String(ss.Config.Sns.TopicArn),
	}
	m.On("ListSubscriptionsByTopic", expectedFirstInput).Return(
		&sns.ListSubscriptionsByTopicOutput{Subscriptions: []*sns.Subscription{sub1, sub2},
			NextToken: aws.String("next")}, nil)

	expectedSecondInput := &sns.ListSubscriptionsByTopicInput{
		NextToken: aws.String("next"),
		TopicArn:  aws.String(ss.Config.Sns.TopicArn),
	}
	m.On("ListSubscriptionsByTopic", expectedSecondInput).Return(
		&sns.ListSubscriptionsByTopicOutput{Subscriptions: []*sns.Subscription{sub3, sub4, sub5}}, nil)

	unsubList, err := ss.ListSubscriptionsByMatchingEndpoint()

	assert.Nil(err)
	require.NotNil(unsubList)
	assert.Equal(3, unsubList.Len())
	item := unsubList.Front()
	assert.Equal("test2", item.Value.(string))
	item = item.Next()
	assert.Equal("test3", item.Value.(string))
	item = item.Next()
	assert.Equal("test5", item.Value.(string))

	m.AssertExpectations(t)
}

func TestListSubscriptionsByMatchingEndpointAWSErr(t *testing.T) {
	require := require.New(t)

	ss, m, _, _ := SetUpTestSNSServer(t)

	// mocking SNS ListSubscriptionsByTopic response to empty list
	m.On("ListSubscriptionsByTopic", mock.AnythingOfType("*sns.ListSubscriptionsByTopicInput")).Return(
		&sns.ListSubscriptionsByTopicOutput{Subscriptions: []*sns.Subscription{}}, fmt.Errorf("%s", "SNS error"))

	unsubList, err := ss.ListSubscriptionsByMatchingEndpoint()

	require.NotNil(err)
	require.Nil(unsubList)
}
