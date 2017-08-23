package webhook

import (
	"encoding/json"
	AWS "github.com/Comcast/webpa-common/webhook/aws"
	"github.com/aws/aws-sdk-go/service/sns"
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

const TEST_UNIX_TIME = 1503357402

func testNow() time.Time {
	return time.Unix(TEST_UNIX_TIME, 0)
}

func TestSubArnError(t *testing.T) {
	n, m, _, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	f.Notifier = n

	assert := assert.New(t)
	expectedSubArn := "pending confirmation"

	// mocking SNS subscribe response
	m.On("Subscribe", mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
		SubscriptionArn: &expectedSubArn}, nil)

	_, handler := f.NewRegistryAndHandler()

	f.Initialize(r, nil, handler, nil, testNow)

	ts := httptest.NewServer(r)

	f.PrepareAndStart()

	time.Sleep(1 * time.Second)

	// AWS SubscriptionConfirmation message is not recevied or delayed
	// causing subscription error during notification

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(AWS.NOTIFY_HOOK_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	errMsg := new(AWS.ErrResp)
	errResp, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusInternalServerError, errMsg.Code)
	assert.Equal(errMsg.Message, "SubscriptionARN does not match")
	assert.Equal(http.StatusInternalServerError, res.StatusCode)

	m.AssertExpectations(t)

}

func TestNotificationBeforeInitialize(t *testing.T) {
	n, _, _, r := AWS.SetUpTestNotifier()

	f, _ := NewFactory(nil)
	f.Notifier = n

	assert := assert.New(t)

	_, handler := f.NewRegistryAndHandler()

	f.Initialize(r, nil, handler, nil, testNow)

	ts := httptest.NewServer(r)

	// SubscriptionArn is not initialized and is nil. SubConf not yet received
	// mocking SNS Publish response

	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL+"/api/v2/aws/sns/"+strconv.Itoa(TEST_UNIX_TIME), strings.NewReader(AWS.NOTIFY_HOOK_MSG))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	req.Header.Add("x-amz-sns-subscription-arn", "testSubscriptionArn")

	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(http.StatusInternalServerError, res.StatusCode)
	errMsg := new(AWS.ErrResp)
	errResp, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal([]byte(errResp), errMsg)

	assert.Equal(http.StatusInternalServerError, errMsg.Code)
	assert.Equal(errMsg.Message, "SubscriptionARN does not match")

}
