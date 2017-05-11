package webhook

import (
	AWS "github.com/Comcast/webpa-common/webhook/aws"
	"github.com/gorilla/mux"
	"testing"
	"net/http/httptest"
	"net/url"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/aws/aws-sdk-go/service/sns"
	"time"
	"strings"
	"net/http"
	"fmt"
)

func SetUpTestNotifier() (AWS.Notifier, *AWS.MockSVC, *mux.Router)  {
	
	v := AWS.SetUpTestViperInstance(AWS.TEST_AWS_CONFIG)
	
	awsCfg, _ := AWS.NewAWSConfig(v.Sub(AWS.AWSKey))
	m := &AWS.MockSVC{}
	
	ss := &AWS.SNSServer{
		Config: *awsCfg,
		SVC: m,
	}
	
	r := mux.NewRouter()
	
	return ss, m, r
}

func TestNotifierReadyFlow(t *testing.T) {
	assert  := assert.New(t)
	expectedSubArn := "pending confirmation"
	confSubArn := "testSubConf"
	
	n,m,r := SetUpTestNotifier()
	
	// mocking SNS subscribe response
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
													SubscriptionArn: &expectedSubArn},nil)
	
	selfURL := &url.URL{
		Scheme:   "http",
		Host:     "127.0.0.1:8090",
	}
	f,_ := NewFactory(nil)
	f.Notifier = n
	
	_, handler := f.NewListAndHandler()
	
	f.Initialize(r,selfURL,handler, nil)
	
	ts := httptest.NewServer(r)
	
	subConfUrl := fmt.Sprintf("%s%s", ts.URL,"/api/v2/aws/sns")
	
	// Mocking AWS SubscriptionConfirmation POST call using http client
	req := httptest.NewRequest("POST", subConfUrl, strings.NewReader(AWS.TEST_SUB_MSG))
	req.Header.Add("x-amz-sns-message-type","SubscriptionConfirmation")
	
	// mocking SNS ConfirmSubscription response
	m.On("ConfirmSubscription",mock.AnythingOfType("*sns.ConfirmSubscriptionInput")).Return(&sns.ConfirmSubscriptionOutput{
													SubscriptionArn: &confSubArn},nil)
	
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
}