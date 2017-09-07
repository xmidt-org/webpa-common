package aws

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_AWS_CFG_ERR = `{
                        "aws": {
                                "accessKey": "accessKey",
                                "secretKey": "secretKey",
                                "env": "cd",
                                "sns" : {
                                        "region" : "us-east-1",
                                        "protocol" : "https", 
                                        "urlPath" : "/api"
                                }
		                }
		          }`
	TEST_AWS_CFG = `{
	"aws": {
        "accessKey": "accessKey",
        "secretKey": "secretKey",
        "env": "test",
        "sns" : {
	        "region" : "us-east-1",
            "protocol" : "http",
			"topicArn" : "arn:aws:sns:us-east-1:1234:test-topic", 
			"urlPath" : "/sns/"
    } } }`
)

func TestNewSNSServerSuccess(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	v := SetUpTestViperInstance(TEST_AWS_CONFIG)
	ss, err := NewSNSServer(v)

	require.NotNil(ss)
	require.Nil(err)
	assert.NotNil(ss.Config)
	assert.NotNil(ss.SVC)
	assert.NotNil(ss.SNSValidator)
	assert.Equal("test", ss.Config.Env)
	assert.Equal("us-east-1", ss.Config.Sns.Region)
}

func TestNewSNSServerAWSConfigError(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	v := SetUpTestViperInstance(TEST_AWS_CFG_ERR)
	ss, err := NewSNSServer(v)

	require.NotNil(err)
	assert.Nil(ss)
}

func TestNewSNSServerViperNil(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ss, err := NewSNSServer(nil)

	require.Nil(err)
	require.NotNil(ss)
	assert.NotNil(ss.Config)
	assert.Equal(ss.Config.AccessKey, "test-accessKey")
	assert.Equal(ss.Config.Sns.TopicArn, "arn:aws:sns:us-east-1:1234:test-topic")
	assert.NotNil(ss.SNSValidator)
}

func TestNewNotifierViperNil(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ss, err := NewNotifier(nil)

	require.Nil(err)
	assert.NotNil(ss)
}

func TestNewNotifierViperNotNil(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	v := SetUpTestViperInstance(TEST_AWS_CONFIG)
	ss, err := NewNotifier(v)

	require.Nil(err)
	assert.NotNil(ss)
}

func TestSubscribeSelfURL_Nil(t *testing.T) {

	// SNSServer initialized with nil selfurl
	ss, m, _, _ := SetUpTestSNSServer(t)
	urlPath := fmt.Sprint("http://host:port/api/v2/aws/sns/", TEST_UNIX_TIME)
	expectedInput := &sns.SubscribeInput{
		Protocol: aws.String("http"),
		TopicArn: aws.String(ss.Config.Sns.TopicArn),
		Endpoint: aws.String(urlPath),
	}
	m.On("Subscribe", expectedInput).Return(&sns.SubscribeOutput{}, fmt.Errorf("%s", "Unreachable"))

	ss.PrepareAndStart()

	m.AssertExpectations(t)
}

func TestInitialize_SNSUrlPathWithTimestamp(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	v := SetUpTestViperInstance(TEST_AWS_CONFIG)
	ss, _ := NewSNSServer(v)
	selfUrl := &url.URL{
		Scheme: "http",
		Host:   "host-test:port",
	}

	ss.Initialize(nil, selfUrl, nil, nil, func() time.Time { return time.Unix(TEST_UNIX_TIME, 0) })

	require.NotNil(ss.errorLog)
	require.NotNil(ss.debugLog)
	assert.Equal(fmt.Sprint(ss.Config.Sns.UrlPath, "/", TEST_UNIX_TIME), selfUrl.Path)
	assert.Equal("http://host-test:port/api/v2/aws/sns/1503357402", ss.SelfUrl.String())
}

func TestInitialize_SNSUrlPathWithSlash(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	v := SetUpTestViperInstance(TEST_AWS_CFG)
	ss, _ := NewSNSServer(v)
	selfUrl := &url.URL{
		Scheme: "http",
		Host:   "host-test:port",
	}

	ss.Initialize(nil, selfUrl, nil, nil, func() time.Time { return time.Unix(TEST_UNIX_TIME, 0) })

	require.NotNil(ss.errorLog)
	require.NotNil(ss.debugLog)
	assert.Equal(fmt.Sprint(ss.Config.Sns.UrlPath, TEST_UNIX_TIME), selfUrl.Path)
	assert.Equal("http://host-test:port/sns/1503357402", ss.SelfUrl.String())
}
