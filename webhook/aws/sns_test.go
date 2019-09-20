package aws

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/xmetrics"
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
	"waitForDns": "2",
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

func handleDnsReqFail(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false
	w.WriteMsg(m)
}

func handleDnsReqSuccess(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false
	rr, err := dns.NewRR("test.service. A 8.8.8.8")
	if err == nil {
		m.Answer = append(m.Answer, rr)
	}
	w.WriteMsg(m)
}

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
	assert.Equal("http://example.com", ss.Config.Sns.AwsEndpoint)
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
	assert.Equal("", ss.Config.Sns.AwsEndpoint)
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
	urlPath := fmt.Sprint("http://host:10000/api/v2/aws/sns/", TEST_UNIX_TIME)
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
		Host:   "host-test:10000",
	}

	registry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics)
	ss.Initialize(nil, selfUrl, "", nil, nil, registry, func() time.Time { return time.Unix(TEST_UNIX_TIME, 0) })

	require.NotNil(ss.errorLog)
	require.NotNil(ss.debugLog)
	assert.Equal(fmt.Sprint(ss.Config.Sns.UrlPath, "/", TEST_UNIX_TIME), selfUrl.Path)
	assert.Equal("http://host-test:10000/api/v2/aws/sns/1503357402", ss.SelfUrl.String())
}

func TestInitialize_SNSUrlPathWithSlash(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	v := SetUpTestViperInstance(TEST_AWS_CFG)
	ss, _ := NewSNSServer(v)
	selfUrl := &url.URL{
		Scheme: "http",
		Host:   "host-test:10000",
	}

	registry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics)
	ss.Initialize(nil, selfUrl, "", nil, nil, registry, func() time.Time { return time.Unix(TEST_UNIX_TIME, 0) })

	require.NotNil(ss.errorLog)
	require.NotNil(ss.debugLog)
	assert.Equal(fmt.Sprint(ss.Config.Sns.UrlPath, TEST_UNIX_TIME), selfUrl.Path)
	assert.Equal("http://host-test:10000/sns/1503357402", ss.SelfUrl.String())
}

func TestDnsReadyFail(t *testing.T) {
	assert := assert.New(t)

	v := SetUpTestViperInstance(TEST_AWS_CFG)
	ss, err := NewNotifier(v)

	assert.Nil(err)
	assert.NotNil(ss)

	// create mock DNS server
	dns.HandleFunc("/", handleDnsReqFail)
	defer func() {
		dns.DefaultServeMux = dns.NewServeMux()
	}()
	dnsServer := &dns.Server{Addr: ":5079", Net: "tcp"}
	go func() {
		err = dnsServer.ListenAndServe()
		assert.Nil(err)
	}()
	defer dnsServer.Shutdown()

	selfUrl := &url.URL{
		Scheme: "http",
		Host:   "host:10000",
	}
	registry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics)
	ss.Initialize(nil, selfUrl, "localhost:5079", nil, nil, registry, func() time.Time { return time.Unix(TEST_UNIX_TIME, 0) })

	err = ss.DnsReady()

	assert.NotNil(err)
}

/*func TestDnsReadySuccess(t *testing.T) {
	assert := assert.New(t)

	v := SetUpTestViperInstance(TEST_AWS_CFG)
	ss, err := NewNotifier(v)

	assert.Nil(err)
	assert.NotNil(ss)

	// create mock DNS server
	dns.HandleFunc("host.", handleDnsReqSuccess)
	defer func() {
		dns.DefaultServeMux = dns.NewServeMux()
	}()
	dnsServer := &dns.Server{Addr: ":5079", Net: "tcp"}
	go func() {
		dnsServer.ListenAndServe()
	}()
	defer dnsServer.Shutdown()

	selfUrl := &url.URL{
		Scheme: "http",
		Host:   "host:10000",
	}
	registry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics)
	ss.Initialize(nil, selfUrl, "localhost:5079", nil, nil, registry, func() time.Time { return time.Unix(TEST_UNIX_TIME, 0) })

	err = ss.DnsReady()

	assert.Nil(err)
}*/

func TestDnsReadyNoSOASuccess(t *testing.T) {
	assert := assert.New(t)

	v := SetUpTestViperInstance(TEST_AWS_CFG)
	ss, err := NewNotifier(v)

	assert.Nil(err)
	assert.NotNil(ss)

	selfUrl := &url.URL{
		Scheme: "http",
		Host:   "host:10000",
	}
	registry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics)
	ss.Initialize(nil, selfUrl, "", nil, nil, registry, func() time.Time { return time.Unix(TEST_UNIX_TIME, 0) })

	err = ss.DnsReady()

	assert.Nil(err)
}
