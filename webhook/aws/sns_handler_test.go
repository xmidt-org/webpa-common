package aws

import (
	"testing"
	"github.com/gorilla/mux"
	"time"
	//"net/http/httptest"
	//"strings"
	"github.com/stretchr/testify/assert"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/mock"
	"net/url"
	"fmt"
)

func SetUpTestSNSServer() (*SNSServer, *mockSVC, *mux.Router)  {
	
	v := SetUpTestViperInstance(TEST_AWS_CONFIG)
	
	awsCfg, err := NewAWSConfig(v.Sub(AWSKey))
	m := &mockSVC{}
	
	ss := &SNSServer{
		Config: *awsCfg,
		SVC: m,
	}
	fmt.Println(ss)
	fmt.Println(err)
	
	selfURL := &url.URL{
		Scheme:   "http",
		Host:     "127.0.0.1:8090",
	}
	
	r := mux.NewRouter()
	ss.Initialize(r, selfURL, nil, nil)
	
	return ss, m, r
}

func TestSubscribeSuccess(t *testing.T) {
	
	assert  := assert.New(t)
	expectedSubArn := "pending confirmation"
	
	ss, m, _ := SetUpTestSNSServer()
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{
													SubscriptionArn: &expectedSubArn},nil)
	ss.PrepareAndStart()
	
	m.AssertExpectations(t)
	
	// wait such that listenSubscriptionData go routine will update the SubscriptionArn value
	time.Sleep(1*time.Second)
	
	assert.Equal(ss.subscriptionArn.Load().(string), expectedSubArn)	
}

func TestSubscribeError(t *testing.T) {
	
	assert  := assert.New(t)
	
	ss, m, _ := SetUpTestSNSServer()
	m.On("Subscribe",mock.AnythingOfType("*sns.SubscribeInput")).Return(&sns.SubscribeOutput{},
		fmt.Errorf("%s", "InvalidClientTokenId"))
	
	ss.PrepareAndStart()
	
	m.AssertExpectations(t)
	
	// wait such that listenSubscriptionData will update the SubscriptionArn value
	time.Sleep(1*time.Second)
	
	assert.Nil(ss.subscriptionArn.Load())
	
}