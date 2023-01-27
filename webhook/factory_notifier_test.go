package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	// nolint:staticcheck
	AWS "github.com/xmidt-org/webpa-common/v2/webhook/aws"
	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

func testNotifierReady(t *testing.T, m *AWS.MockSVC, mv *AWS.MockValidator, r *mux.Router, f *Factory) (*httptest.Server, Registry) {
	assert := assert.New(t)
	// nolint:goconst
	expectedSubArn := "pending confirmation"
	confSubArn := "testSubscriptionArn"

	// mocking SNS subscribe response
	// nolint: typecheck
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)

	metricsRegistry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics, AWS.Metrics)
	webhookMetrics := ApplyMetricsData(metricsRegistry)
	// nolint: typecheck
	registry, handler := f.NewRegistryAndHandler(webhookMetrics)
	// nolint: typecheck
	f.Initialize(r, nil, "", handler, nil, AWS.ApplyMetricsData(metricsRegistry), testNow)

	ts := httptest.NewServer(r)

	subConfUrl := fmt.Sprintf("%s%s/%d", ts.URL, "/api/v2/aws/sns", TEST_UNIX_TIME)

	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(AWS.TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type", "SubscriptionConfirmation")

	// mocking SNS ConfirmSubscription response
	// nolint: typecheck
	m.On("ConfirmSubscription", mock.AnythingOfType("*sns.ConfirmSubscriptionInput")).Return(&sns.ConfirmSubscriptionOutput{
		SubscriptionArn: &confSubArn}, nil)

	// mocking SNS ListSubscriptionsByTopic response to empty list
	// nolint: typecheck
	m.On("ListSubscriptionsByTopic", mock.AnythingOfType("*sns.ListSubscriptionsByTopicInput")).Return(
		&sns.ListSubscriptionsByTopicOutput{Subscriptions: []*sns.Subscription{}}, nil)

	// nolint: typecheck
	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil).Once()

	// nolint: typecheck
	f.PrepareAndStart()

	// nolint: typecheck
	subValid := f.ValidateSubscriptionArn("")

	assert.Equal(subValid, false)

	req.RequestURI = ""
	// nolint:bodyclose
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusOK, res.StatusCode)

	time.Sleep(1 * time.Second)
	// nolint: typecheck
	subConfValid := f.ValidateSubscriptionArn(confSubArn)

	assert.Equal(true, subConfValid)

	// nolint: typecheck
	m.AssertExpectations(t)
	// nolint: typecheck
	mv.AssertExpectations(t)

	return ts, registry
}

func TestNotifierReadyFlow(t *testing.T) {

	n, m, mv, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	// nolint: typecheck
	f.Notifier = n
	// nolint: typecheck
	f.m = &monitor{}

	testNotifierReady(t, m, mv, r, f)
}

func TestNotifierReadyValidateErr(t *testing.T) {
	assert := assert.New(t)

	n, m, mv, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	// nolint: typecheck
	f.Notifier = n

	// nolint:goconst
	expectedSubArn := "pending confirmation"
	confSubArn := "testSubscriptionArn"

	// mocking SNS subscribe response
	// nolint: typecheck
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)

	metricsRegistry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics, AWS.Metrics)
	webhookMetrics := ApplyMetricsData(metricsRegistry)
	// nolint: typecheck
	_, handler := f.NewRegistryAndHandler(webhookMetrics)
	// nolint: typecheck
	f.Initialize(r, nil, "", handler, nil, AWS.ApplyMetricsData(metricsRegistry), testNow)

	ts := httptest.NewServer(r)

	subConfUrl := fmt.Sprintf("%s%s/%d", ts.URL, "/api/v2/aws/sns", TEST_UNIX_TIME)

	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(AWS.TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type", "SubscriptionConfirmation")

	// nolint: typecheck
	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(false,
		fmt.Errorf("%s", AWS.SNS_VALIDATION_ERR))

	// nolint: typecheck
	f.PrepareAndStart()

	// nolint: typecheck
	subValid := f.ValidateSubscriptionArn("")

	assert.Equal(false, subValid)

	req.RequestURI = ""
	// nolint:bodyclose
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusBadRequest, res.StatusCode)
	errMsg := new(AWS.ErrResp)
	errResp, _ := io.ReadAll(res.Body)
	// nolint:unconvert
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusBadRequest, errMsg.Code)
	assert.Equal(AWS.SNS_VALIDATION_ERR, errMsg.Message)

	// nolint: typecheck
	subConfValid := f.ValidateSubscriptionArn(confSubArn)
	assert.Equal(false, subConfValid)

	// nolint: typecheck
	m.AssertExpectations(t)
	// nolint: typecheck
	mv.AssertExpectations(t)
}

func TestNotifierPublishFlow(t *testing.T) {
	assert := assert.New(t)
	n, m, mv, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	// setting to mocked Notifier instance
	// nolint: typecheck
	f.Notifier = n

	ts, registry := testNotifierReady(t, m, mv, r, f)

	// mocking SNS Publish response
	// nolint: typecheck
	m.On("Publish", mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{}, nil)

	// nolint: typecheck
	f.PublishMessage(AWS.TEST_HOOK)

	time.Sleep(1 * time.Second)

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(AWS.NOTIFY_HOOK_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	// nolint: typecheck
	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil)

	req.RequestURI = ""
	// nolint:bodyclose
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

	// nolint: typecheck
	m.AssertExpectations(t)
	// nolint: typecheck
	mv.AssertExpectations(t)
}

func TestNotifierPublishTopicArnMismatch(t *testing.T) {

	assert := assert.New(t)
	n, m, mv, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	// setting to mocked Notifier instance
	// nolint: typecheck
	f.Notifier = n

	ts, registry := testNotifierReady(t, m, mv, r, f)

	// mocking SNS Publish response
	// nolint: typecheck
	m.On("Publish", mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{}, nil)

	// nolint: typecheck
	f.PublishMessage(AWS.TEST_HOOK)

	time.Sleep(1 * time.Second)

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(AWS.TEST_NOTIF_ERR_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	// nolint: typecheck
	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil)

	req.RequestURI = ""
	// nolint:bodyclose
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusBadRequest, res.StatusCode)
	errMsg := new(AWS.ErrResp)
	errResp, _ := io.ReadAll(res.Body)

	// nolint:unconvert
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusBadRequest, errMsg.Code)
	assert.Equal("TopicArn does not match", errMsg.Message)
	assert.Equal(0, registry.m.list.Len())

	// nolint: typecheck
	m.AssertExpectations(t)
	// nolint: typecheck
	mv.AssertExpectations(t)

}

func TestNotifierPublishValidateErr(t *testing.T) {

	assert := assert.New(t)
	n, m, mv, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	// setting to mocked Notifier instance
	// nolint: typecheck
	f.Notifier = n

	ts, registry := testNotifierReady(t, m, mv, r, f)

	// mocking SNS Publish response
	// nolint: typecheck
	m.On("Publish", mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{}, nil)

	// nolint: typecheck
	f.PublishMessage(AWS.TEST_HOOK)

	time.Sleep(1 * time.Second)

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(AWS.TEST_NOTIF_ERR_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	// nolint: typecheck
	mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(false,
		fmt.Errorf("%s", AWS.SNS_VALIDATION_ERR))

	req.RequestURI = ""
	// nolint:bodyclose
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusBadRequest, res.StatusCode)
	errMsg := new(AWS.ErrResp)
	errResp, _ := io.ReadAll(res.Body)
	// nolint:unconvert
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusBadRequest, errMsg.Code)
	assert.Equal(AWS.SNS_VALIDATION_ERR, errMsg.Message)
	assert.Equal(0, registry.m.list.Len())

	// nolint: typecheck
	m.AssertExpectations(t)
	// nolint: typecheck
	mv.AssertExpectations(t)

}
