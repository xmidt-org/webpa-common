package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	AWS "github.com/xmidt-org/webpa-common/webhook/aws"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

type Testing []struct {
	name     string
	HookTest HookTest
	// NotifyHookMsg NotifyHookMsg
	NotifyHookMsg   string
	returnStatus    int
	expectedURL     string
	expectedListLen int
}

type HookTest struct {
	Config  Config   `json:"config"`
	Matcher Matcher  `json:"matcher"`
	Events  []string `json:"events"`
}

type Config struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
}

type Matcher struct {
	DeviceID []string `json:"device_id"`
}

type NotifyHookMsg struct {
	Type              string            `json:"Type"`
	MessageID         string            `json:"MessageId"`
	Token             string            `json:"Token"`
	TopicArn          string            `json:"TopicArn"`
	Subject           string            `json:"Subject"`
	Message           string            `json:"Message"`
	Timestamp         string            `json:"Timestamp"`
	SignatureVersion  string            `json:"SignatureVersion"`
	Signature         string            `json:"Signature"`
	SigningCertURL    string            `json:"SigningCertURL"`
	SubscribeURL      string            `json:"SubscribeURL"`
	UnsubscribeURL    string            `json:"UnsubscribeURL"`
	MessageAttributes MessageAttributes `json:"MessageAttributes"`
}

type MessageAttributes struct {
	ScytaleEnv ScytaleEnv `json:"scytale.env"`
}

type ScytaleEnv struct {
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

var (
	NotifErrMsgTest = NotifyHookMsg{
		Type:             "Notification",
		MessageID:        "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
		TopicArn:         "Invalid-topic",
		Subject:          "My First Message",
		Message:          "Hello world!",
		Timestamp:        "2012-05-02T00:54:06.655Z",
		SignatureVersion: "1",
		Signature:        "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
		SigningCertURL:   "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
		UnsubscribeURL:   "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96",
		MessageAttributes: MessageAttributes{
			ScytaleEnv{
				Type:  "String",
				Value: "test",
			}},
	}

	SubMsgTest = NotifyHookMsg{
		Type:             "SubscriptionConfirmation",
		MessageID:        "165545c9-2a5c-472c-8df2-7ff2be2b3b1b",
		Token:            "2336412f37fb687f5d51e6e241d09c805a5a57b30d712f794cc5f6a988666d92768dd60a747ba6f3beb71854e285d6ad02428b09ceece29417f1f02d609c582afbacc99c583a916b9981dd2728f4ae6fdb82efd087cc3b7849e05798d2d2785c03b0879594eeac82c01f235d0e717736",
		TopicArn:         "arn:aws:sns:us-east-1:1234:test-topic",
		Message:          "You have chosen to subscribe to the topic arn:aws:sns:us-east-1:1234:test-topic.\nTo confirm the subscription, visit the SubscribeURL included in this message.",
		SubscribeURL:     "https://sns.us-west-2.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn:aws:sns:us-west-2:123456789012:MyTopic&Token=2336412f37fb687f5d51e6e241d09c805a5a57b30d712f794cc5f6a988666d92768dd60a747ba6f3beb71854e285d6ad02428b09ceece29417f1f02d609c582afbacc99c583a916b9981dd2728f4ae6fdb82efd087cc3b7849e05798d2d2785c03b0879594eeac82c01f235d0e717736",
		Timestamp:        "2012-04-26T20:45:04.751Z",
		SignatureVersion: "1",
		Signature:        "EXAMPLEpH+DcEwjAPg8O9mY8dReBSwksfg2S7WKQcikcNKWLQjwu6A4VbeS0QHVCkhRS7fUQvi2egU3N858fiTDN6bkkOxYDVrY0Ad8L10Hs3zH81mtnPk5uvvolIC1CXGu43obcgFxeL3khZl8IKvO61GWB6jI9b5+gLPoBc1Q=",
		SigningCertURL:   "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
	}
)

func testNotifierReady(t *testing.T, m *AWS.MockSVC, mv *AWS.MockValidator, r *mux.Router, f *Factory) (*httptest.Server, Registry) {
	assert := assert.New(t)
	expectedSubArn := "pending confirmation"
	confSubArn := "testSubscriptionArn"

	// mocking SNS subscribe response
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)

	metricsRegistry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics, AWS.Metrics)
	registry, handler := f.NewRegistryAndHandler(metricsRegistry)
	f.Initialize(r, nil, "", handler, nil, metricsRegistry, testNow)

	ts := httptest.NewServer(r)

	subConfUrl := fmt.Sprintf("%s%s/%d", ts.URL, "/api/v2/aws/sns", TEST_UNIX_TIME)

	// Mocking AWS SubscriptionConfirmation POST call using http client
	subMsgJSON, err := json.Marshal(SubMsgTest)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(string(subMsgJSON)))
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
	f.m = &monitor{}

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

	metricsRegistry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics, AWS.Metrics)
	_, handler := f.NewRegistryAndHandler(metricsRegistry)
	f.Initialize(r, nil, "", handler, nil, metricsRegistry, testNow)

	ts := httptest.NewServer(r)

	subConfUrl := fmt.Sprintf("%s%s/%d", ts.URL, "/api/v2/aws/sns", TEST_UNIX_TIME)

	// Mocking AWS SubscriptionConfirmation POST call using http client
	subMsgJSON, err := json.Marshal(SubMsgTest)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(string(subMsgJSON)))
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

	tests := &Testing{
		{
			name: "http+ip",
			HookTest: HookTest{
				Config: Config{
					URL:         "http://127.0.0.1:8080/test",
					ContentType: "json",
					Secret:      "",
				},
				Matcher: Matcher{
					DeviceID: []string{".*"},
				},
				Events: []string{
					"transaction-status",
					"SYNC_NOTIFICATION"},
			},
			NotifyHookMsg: `{
				"Type" : "Notification",
				"MessageId" : "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
				"TopicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
				"Subject" : "My First Message",
				"Message" : "[{\"config\":{\"url\":\"http://127.0.0.1:8080/test\",\"content_type\":\"json\",\"secret\":\"\"},\"matcher\":{\"device_id\":[\".*\"]},\"events\":[\"transaction-status\",\"SYNC_NOTIFICATION\"]}]",
				"Timestamp" : "2012-05-02T00:54:06.655Z",
				"SignatureVersion" : "1",
				"Signature" : "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
				"SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
				"UnsubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96",
				"MessageAttributes" : {
				"scytale.env" : {"Type":"String","Value":"test"}
				} }`,
			returnStatus:    http.StatusOK,
			expectedURL:     "http://127.0.0.1:8080/test",
			expectedListLen: 1,
		},
		{
			name: "https+ip",
			HookTest: HookTest{
				Config: Config{
					URL:         "https://127.0.0.1:8080/test",
					ContentType: "json",
					Secret:      "",
				},
				Matcher: Matcher{
					DeviceID: []string{".*"},
				},
				Events: []string{
					"transaction-status",
					"SYNC_NOTIFICATION"},
			},
			NotifyHookMsg: `{
				"Type" : "Notification",
				"MessageId" : "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
				"TopicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
				"Subject" : "My First Message",
				"Message" : "[{\"config\":{\"url\":\"https://127.0.0.1:8080/test\",\"content_type\":\"json\",\"secret\":\"\"},\"matcher\":{\"device_id\":[\".*\"]},\"events\":[\"transaction-status\",\"SYNC_NOTIFICATION\"]}]",
				"Timestamp" : "2012-05-02T00:54:06.655Z",
				"SignatureVersion" : "1",
				"Signature" : "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
				"SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
				"UnsubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96",
				"MessageAttributes" : {
				"scytale.env" : {"Type":"String","Value":"test"}
				} }`,
			returnStatus:    http.StatusBadRequest,
			expectedURL:     "https://127.0.0.1:8080/test",
			expectedListLen: 0,
		},
		{
			name: "https+dns",
			HookTest: HookTest{
				Config: Config{
					URL:         "https://example/test",
					ContentType: "json",
					Secret:      "",
				},
				Matcher: Matcher{
					DeviceID: []string{".*"},
				},
				Events: []string{
					"transaction-status",
					"SYNC_NOTIFICATION"},
			},
			NotifyHookMsg: `{
				"Type" : "Notification",
				"MessageId" : "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
				"TopicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
				"Subject" : "My First Message",
				"Message" : "[{\"config\":{\"url\":\"https://example/test\",\"content_type\":\"json\",\"secret\":\"\"},\"matcher\":{\"device_id\":[\".*\"]},\"events\":[\"transaction-status\",\"SYNC_NOTIFICATION\"]}]",
				"Timestamp" : "2012-05-02T00:54:06.655Z",
				"SignatureVersion" : "1",
				"Signature" : "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
				"SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
				"UnsubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96",
				"MessageAttributes" : {
				"scytale.env" : {"Type":"String","Value":"test"}
				} }`,
			returnStatus:    http.StatusOK,
			expectedURL:     "https://example/test",
			expectedListLen: 1,
		},
	}

	for _, test := range *tests {
		t.Run(test.name, func(t *testing.T) {
			hookTestJSON, err := json.Marshal(test.HookTest)
			if err != nil {
				t.Fatal(err)
			}
			notifyHookMsgJSON, err := json.Marshal(test.NotifyHookMsg)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println(string(notifyHookMsgJSON))

			assert := assert.New(t)
			n, m, mv, r := AWS.SetUpTestNotifier()

			f, _ := NewFactory(nil)
			// setting to mocked Notifier instance
			f.Notifier = n

			ts, registry := testNotifierReady(t, m, mv, r, f)

			// mocking SNS Publish response
			m.On("Publish", mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{}, nil)

			//f.PublishMessage(AWS.TEST_HOOK)
			f.PublishMessage(string(hookTestJSON))

			time.Sleep(1 * time.Second)

			// Mocking SNS Notification POST call
			req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(test.NotifyHookMsg))
			req.Header.Add("x-amz-sns-message-type", "Notification")
			req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

			mv.On("Validate", mock.AnythingOfType("*aws.SNSMessage")).Return(true, nil)

			req.RequestURI = ""
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(test.returnStatus, res.StatusCode)

			time.Sleep(1 * time.Second)

			assert.Equal(test.expectedListLen, registry.m.list.Len())

			// Assert the notification webhook W received matches the one that was sent in publish message

			if registry.m.list.Len() > 0 {
				hook := registry.m.list.Get(0)

				assert.Equal([]string{"transaction-status", "SYNC_NOTIFICATION"}, hook.Events)
				assert.Equal(test.expectedURL, hook.Config.URL)
				assert.Equal([]string{".*"}, hook.Matcher.DeviceId)

				m.AssertExpectations(t)
				mv.AssertExpectations(t)
			}

		})
	}

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
	notifErrJSON, err := json.Marshal(NotifErrMsgTest)
	if err != nil {
		t.Error(err)
	}
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(string(notifErrJSON)))
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
	notifErrJSON, err := json.Marshal(NotifErrMsgTest)
	if err != nil {
		t.Error(err)
	}
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(string(notifErrJSON)))
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
