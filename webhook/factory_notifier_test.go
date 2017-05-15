package webhook

import (
	AWS "github.com/Comcast/webpa-common/webhook/aws"
	"github.com/gorilla/mux"
	"testing"
	"net/http/httptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/aws/aws-sdk-go/service/sns"
	"time"
	"strings"
	"net/http"
	"fmt"
	"io/ioutil"
	"encoding/json"
)

func testNotifierReady(t *testing.T, m *AWS.MockSVC, mv *AWS.MockValidator, r *mux.Router, f *Factory) (*httptest.Server, List) {
	assert  := assert.New(t)
	expectedSubArn := "pending confirmation"
	confSubArn := "testSubscriptionArn"
	
	// mocking SNS subscribe response
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
													SubscriptionArn: &expectedSubArn},nil)
	
	list, handler := f.NewListAndHandler()
	
	f.Initialize(r,nil,handler, nil)
	
	ts := httptest.NewServer(r)
	
	subConfUrl := fmt.Sprintf("%s%s", ts.URL,"/api/v2/aws/sns")
	
	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(AWS.TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type","SubscriptionConfirmation")
	
	// mocking SNS ConfirmSubscription response
	m.On("ConfirmSubscription",mock.AnythingOfType("*sns.ConfirmSubscriptionInput")).Return(&sns.ConfirmSubscriptionOutput{
													SubscriptionArn: &confSubArn},nil)
	
	mv.On("Validate",mock.AnythingOfType("*aws.SNSMessage")).Return(true,nil).Once()
	
	f.PrepareAndStart()
	
	time.Sleep(1*time.Second)
	
	subValid := f.ValidateSubscriptionArn(expectedSubArn)
	
	assert.Equal(subValid, true)
	
	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(res.StatusCode,http.StatusOK)
	
	time.Sleep(1*time.Second)
	subConfValid := f.ValidateSubscriptionArn(confSubArn)
	
	assert.Equal(subConfValid, true)
	
	m.AssertExpectations(t)
	mv.AssertExpectations(t)
	
	return ts, list
}

func TestNotifierReadyFlow(t *testing.T) {
	
	n,m,mv,r := AWS.SetUpTestNotifier()
	
	f,_ := NewFactory(nil)
	f.Notifier = n
	
	testNotifierReady(t,m,mv,r,f)
}

func TestNotifierReadyValidateErr(t *testing.T) {
	assert  := assert.New(t)
	
	n,m,mv,r := AWS.SetUpTestNotifier()
	
	f,_ := NewFactory(nil)
	f.Notifier = n
	
	expectedSubArn := "pending confirmation"
	confSubArn := "testSubscriptionArn"
	
	// mocking SNS subscribe response
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
													SubscriptionArn: &expectedSubArn},nil)
	
	_, handler := f.NewListAndHandler()
	
	f.Initialize(r,nil,handler, nil)
	
	ts := httptest.NewServer(r)
	
	subConfUrl := fmt.Sprintf("%s%s", ts.URL,"/api/v2/aws/sns")
	
	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(AWS.TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type","SubscriptionConfirmation")
	
	mv.On("Validate",mock.AnythingOfType("*aws.SNSMessage")).Return(false,
		fmt.Errorf("%s", AWS.SNS_VALIDATION_ERR))
	
	f.PrepareAndStart()
	
	time.Sleep(1*time.Second)
	
	subValid := f.ValidateSubscriptionArn(expectedSubArn)
	
	assert.Equal(subValid, true)
	
	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(res.StatusCode,http.StatusBadRequest)
	errMsg := new(AWS.ErrResp)
	errResp, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal([]byte(errResp), errMsg)
    
    assert.Equal(errMsg.Code,http.StatusBadRequest)
    assert.Equal(errMsg.Message,AWS.SNS_VALIDATION_ERR)
	
	subConfValid := f.ValidateSubscriptionArn(confSubArn)
	assert.Equal(subConfValid, false)
	
	m.AssertExpectations(t)
	mv.AssertExpectations(t)
}

func TestNotifierPublishFlow(t *testing.T) {
	assert  := assert.New(t)
	n,m,mv,r := AWS.SetUpTestNotifier()
	
	f,_ := NewFactory(nil)
	// setting to mocked Notifier instance
	f.Notifier = n
	
	ts, list := testNotifierReady(t,m,mv,r,f)
	
	// mocking SNS Publish response
	m.On("Publish",mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{},nil)
	
	f.PublishMessage(AWS.TEST_HOOK)
	
	time.Sleep(1*time.Second)
	
	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL + "/api/v2/aws/sns", strings.NewReader(AWS.NOTIFY_HOOK_MSG))
	req.Header.Add("x-amz-sns-message-type","Notification")
	req.Header.Add("x-amz-sns-subscription-arn","testSubscriptionArn")
	
	mv.On("Validate",mock.AnythingOfType("*aws.SNSMessage")).Return(true,nil)
	
	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(res.StatusCode,http.StatusOK)
	
	time.Sleep(1*time.Second)
	
	assert.Equal(list.Len(),1)
	
	// Assert the notification webhook W received matches the one that was sent in publish message
	hook := *list.Get(0)
	
	assert.Equal(hook.Events,[]string{"transaction-status","SYNC_NOTIFICATION"} )
	assert.Equal(hook.Config.URL, "http://127.0.0.1:8080/test")
	assert.Equal(hook.Matcher.DeviceId,[]string{".*"})
	
	m.AssertExpectations(t)
	mv.AssertExpectations(t)
}

func TestNotifierPublishTopicArnMismatch(t *testing.T) {
	
	assert  := assert.New(t)
	n,m,mv,r := AWS.SetUpTestNotifier()
	
	f,_ := NewFactory(nil)
	// setting to mocked Notifier instance
	f.Notifier = n
	
	ts, list := testNotifierReady(t,m,mv,r,f)
	
	// mocking SNS Publish response
	m.On("Publish",mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{},nil)
	
	f.PublishMessage(AWS.TEST_HOOK)
	
	time.Sleep(1*time.Second)
	
	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL + "/api/v2/aws/sns", strings.NewReader(AWS.TEST_NOTIF_ERR_MSG))
	req.Header.Add("x-amz-sns-message-type","Notification")
	req.Header.Add("x-amz-sns-subscription-arn","testSubscriptionArn")
	
	mv.On("Validate",mock.AnythingOfType("*aws.SNSMessage")).Return(true,nil)
	
	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(res.StatusCode,http.StatusBadRequest)
	errMsg := new(AWS.ErrResp)
	errResp, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal([]byte(errResp), errMsg)
    
    assert.Equal(errMsg.Code,http.StatusBadRequest)
    assert.Equal(errMsg.Message,"TopicArn does not match")
    assert.Equal(list.Len(),0)
	
	m.AssertExpectations(t)
	mv.AssertExpectations(t)
	
}

func TestNotifierPublishValidateErr(t *testing.T) {
	
	assert  := assert.New(t)
	n,m,mv,r := AWS.SetUpTestNotifier()
	
	f,_ := NewFactory(nil)
	// setting to mocked Notifier instance
	f.Notifier = n
	
	ts, list := testNotifierReady(t,m,mv,r,f)
	
	// mocking SNS Publish response
	m.On("Publish",mock.AnythingOfType("*sns.PublishInput")).Return(&sns.PublishOutput{},nil)
	
	f.PublishMessage(AWS.TEST_HOOK)
	
	time.Sleep(1*time.Second)
	
	// Mocking SNS Notification POST call
	req := httptest.NewRequest("POST", ts.URL + "/api/v2/aws/sns", strings.NewReader(AWS.TEST_NOTIF_ERR_MSG))
	req.Header.Add("x-amz-sns-message-type","Notification")
	req.Header.Add("x-amz-sns-subscription-arn","testSubscriptionArn")
	
	mv.On("Validate",mock.AnythingOfType("*aws.SNSMessage")).Return(false,
		fmt.Errorf("%s", AWS.SNS_VALIDATION_ERR))
	
	req.RequestURI = ""
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(res.StatusCode,http.StatusBadRequest)
	errMsg := new(AWS.ErrResp)
	errResp, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal([]byte(errResp), errMsg)
    
    assert.Equal(errMsg.Code,http.StatusBadRequest)
    assert.Equal(errMsg.Message,AWS.SNS_VALIDATION_ERR)
    assert.Equal(list.Len(),0)
	
	m.AssertExpectations(t)
	mv.AssertExpectations(t)
	
}
