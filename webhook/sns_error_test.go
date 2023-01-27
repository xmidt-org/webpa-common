package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	// nolint:staticcheck
	AWS "github.com/xmidt-org/webpa-common/v2/webhook/aws"
	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

const TEST_UNIX_TIME = 1503357402

func testNow() time.Time {
	return time.Unix(TEST_UNIX_TIME, 0)
}

func TestSubArnError(t *testing.T) {
	n, m, _, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	// nolint: typecheck
	f.Notifier = n

	assert := assert.New(t)
	expectedSubArn := "pending confirmation"

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

	// nolint: typecheck
	f.PrepareAndStart()

	time.Sleep(1 * time.Second)

	// AWS SubscriptionConfirmation message is not recevied or delayed
	// causing subscription error during notification

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(AWS.NOTIFY_HOOK_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	req.RequestURI = ""
	// nolint:bodyclose
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	errMsg := new(AWS.ErrResp)
	errResp, _ := io.ReadAll(res.Body)
	// nolint:unconvert
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusInternalServerError, errMsg.Code)
	assert.Equal(errMsg.Message, "SubscriptionARN does not match")
	assert.Equal(http.StatusInternalServerError, res.StatusCode)

	// nolint: typecheck
	m.AssertExpectations(t)

}

func TestNotificationBeforeInitialize(t *testing.T) {
	n, _, _, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	// nolint: typecheck
	f.Notifier = n

	assert := assert.New(t)

	metricsRegistry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics, AWS.Metrics)
	webhookMetrics := ApplyMetricsData(metricsRegistry)
	// nolint: typecheck
	_, handler := f.NewRegistryAndHandler(webhookMetrics)
	// nolint: typecheck
	f.Initialize(r, nil, "", handler, nil, AWS.ApplyMetricsData(metricsRegistry), testNow)

	ts := httptest.NewServer(r)

	// SubscriptionArn is not initialized and is nil. SubConf not yet received
	// mocking SNS Publish response

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(AWS.NOTIFY_HOOK_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	req.RequestURI = ""
	// nolint:bodyclose
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusInternalServerError, res.StatusCode)
	errMsg := new(AWS.ErrResp)
	errResp, _ := io.ReadAll(res.Body)
	// nolint:unconvert
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusInternalServerError, errMsg.Code)
	assert.Equal(errMsg.Message, "SubscriptionARN does not match")

}
