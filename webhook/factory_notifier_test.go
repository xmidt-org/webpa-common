package webhook

import (
	"encoding/json"
	"fmt"
	AWS "github.com/Comcast/webpa-common/webhook/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testNotifierReady(t *testing.T, m *AWS.MockSVC, mv *AWS.MockValidator, r *mux.Router, f *Factory) (*httptest.Server, Registry) {
	assert := assert.New(t)
	expectedSubArn := "pending confirmation"
	confSubArn := "testSubscriptionArn"

	// mocking SNS subscribe response
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)

	registry, handler := f.NewRegistryAndHandler()

	f.Initialize(r, nil, handler, nil, testNow)

	ts := httptest.NewServer(r)

	subConfUrl := fmt.Sprintf("%s%s/%d", ts.URL, "/api/v2/aws/sns", TEST_UNIX_TIME)

	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(AWS.TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type", "SubscriptionConfirmation")

	// mocking SNS ConfirmSubscription response
	m.On("ConfirmSubscription", mock.AnythingOfType("*sns.ConfirmSubscriptionInput")).Return(&sns.ConfirmSubscriptionOutput{
		SubscriptionArn: &confSubArn}, nil)

	// mocking SNS ListSubscriptionsByTopic response to empty list
	m.On("ListSubscriptionsByTopic", mock.AnythingOfType("*sns.ListSubscriptionsByTopicInput")).Return(
		&sns.ListSubscriptionsByTopicOutput{Subscriptions: []*sns.Subscription{}}, nil)

	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil).Once()

	f.PrepareAndStart()


	subValid := f.ValidateSubscriptionArn("")

	assert.Equal(subValid, false)

	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusOK, res.StatusCode)

	time.Sleep(1 * time.Second)
	subConfValid := f.ValidateSubscriptionArn(confSubArn)

	assert.Equal(true, subConfValid)

	m.AssertExpectations(t)
	mv.AssertExpectations(t)

	return ts, registry
}

func TestNotifierReadyFlow(t *testing.T) {

	n, m, mv, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	f.Notifier = n

	testNotifierReady(t, m, mv, r, f)
}

func TestNotifierReadyValidateErr(t *testing.T) {
	assert := assert.New(t)

	n, m, mv, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	f.Notifier = n

	expectedSubArn := "pending confirmation"
	confSubArn := "testSubscriptionArn"

	// mocking SNS subscribe response
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)

	_, handler := f.NewRegistryAndHandler()

	f.Initialize(r, nil, handler, nil, testNow)

	ts := httptest.NewServer(r)

	subConfUrl := fmt.Sprintf("%s%s/%d", ts.URL, "/api/v2/aws/sns", TEST_UNIX_TIME)

	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(AWS.TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type", "SubscriptionConfirmation")

	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(false,
		fmt.Errorf("%s", AWS.SNS_VALIDATION_ERR))

	f.PrepareAndStart()

	subValid := f.ValidateSubscriptionArn("")

	assert.Equal(false, subValid)

	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusBadRequest, res.StatusCode)
	errMsg := new(AWS.ErrResp)
	errResp, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusBadRequest, errMsg.Code)
	assert.Equal(AWS.SNS_VALIDATION_ERR, errMsg.Message)

	subConfValid := f.ValidateSubscriptionArn(confSubArn)
	assert.Equal(false, subConfValid)

	m.AssertExpectations(t)
	mv.AssertExpectations(t)
}

func TestNotifierPublishFlow(t *testing.T) {
	assert := assert.New(t)
	n, m, mv, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	// setting to mocked Notifier instance
	f.Notifier = n

	ts, registry := testNotifierReady(t, m, mv, r, f)

	// mocking SNS Publish response
	m.On("Publish", mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{}, nil)

	f.PublishMessage(AWS.TEST_HOOK)

	time.Sleep(1 * time.Second)

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(AWS.NOTIFY_HOOK_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil)

	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusOK, res.StatusCode)

	time.Sleep(1 * time.Second)

	assert.Equal(1, registry.m.list.Len())

	// Assert the notification webhook W received matches the one that was sent in publish message
	hook := registry.m.list.Get(0)

	assert.Equal([]string{"transaction-status", "SYNC_NOTIFICATION"}, hook.Events)
	assert.Equal("http://127.0.0.1:8080/test", hook.Config.URL)
	assert.Equal([]string{".*"}, hook.Matcher.DeviceId)

	m.AssertExpectations(t)
	mv.AssertExpectations(t)
}

func TestNotifierPublishTopicArnMismatch(t *testing.T) {

	assert := assert.New(t)
	n, m, mv, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	// setting to mocked Notifier instance
	f.Notifier = n

	ts, registry := testNotifierReady(t, m, mv, r, f)

	// mocking SNS Publish response
	m.On("Publish", mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{}, nil)

	f.PublishMessage(AWS.TEST_HOOK)

	time.Sleep(1 * time.Second)

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(AWS.TEST_NOTIF_ERR_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil)

	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusBadRequest, res.StatusCode)
	errMsg := new(AWS.ErrResp)
	errResp, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusBadRequest, errMsg.Code)
	assert.Equal("TopicArn does not match", errMsg.Message)
	assert.Equal(0, registry.m.list.Len())

	m.AssertExpectations(t)
	mv.AssertExpectations(t)

}

func TestNotifierPublishValidateErr(t *testing.T) {

	assert := assert.New(t)
	n, m, mv, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	// setting to mocked Notifier instance
	f.Notifier = n

	ts, registry := testNotifierReady(t, m, mv, r, f)

	// mocking SNS Publish response
	m.On("Publish", mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{}, nil)

	f.PublishMessage(AWS.TEST_HOOK)

	time.Sleep(1 * time.Second)

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(AWS.TEST_NOTIF_ERR_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(false,
		fmt.Errorf("%s", AWS.SNS_VALIDATION_ERR))

	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusBadRequest, res.StatusCode)
	errMsg := new(AWS.ErrResp)
	errResp, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusBadRequest, errMsg.Code)
	assert.Equal(AWS.SNS_VALIDATION_ERR, errMsg.Message)
	assert.Equal(0, registry.m.list.Len())

	m.AssertExpectations(t)
	mv.AssertExpectations(t)

}
