// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"bytes"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

const (
	TEST_SNS_CFG = `{
	"aws": {
        "accessKey": "test-accessKey",
        "secretKey": "test-secretKey",
        "env": "test",
        "sns" : {
	        "region" : "us-east-1",
            "protocol" : "http",
			"topicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
			"urlPath" : "/sns/"
    } } }`
	NOTIF_MSG = `{
  "Type" : "Notification",
  "MessageId" : "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
  "TopicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
  "Subject" : "My First Message",
  "Message" : "Hello world!",
  "Timestamp" : "2012-05-02T00:54:06.655Z",
  "SignatureVersion" : "1",
  "Signature" : "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
  "SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
  "UnsubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96",
  "MessageAttributes" : {
    "scytale.env" : {"Type":"String","Value":"test"}
  } }`
	TEST_NOTIF_MSG = `{
  "Type" : "Notification",
  "MessageId" : "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
  "TopicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
  "Subject" : "My First Message",
  "Message" : "Hello world!",
  "Timestamp" : "2012-05-02T00:54:06.655Z",
  "SignatureVersion" : "1",
  "Signature" : "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
  "SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
  "UnsubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96",
  "MessageAttributes" : {
    "scytale.env" : {"Type":"String","Value":"Invalid"}
  } }`
	TEST_UNIX_TIME = 1503357402
)

func testNow() time.Time {
	return time.Unix(TEST_UNIX_TIME, 0)
}

func SetUpTestViperInstance(config string) *viper.Viper {

	cfg := bytes.NewBufferString(config)
	v := viper.New()
	v.SetConfigType("json")
	v.ReadConfig(cfg)
	return v
}

func SetUpTestSNSServer(t *testing.T) (*SNSServer, *MockSVC, *MockValidator, *mux.Router) {
	return SetUpTestSNSServerWithChannelSize(t, 50)
}

func SetUpTestSNSServerWithChannelSize(t *testing.T, channelSize int64) (*SNSServer, *MockSVC, *MockValidator, *mux.Router) {

	v := SetUpTestViperInstance(TEST_AWS_CONFIG)

	awsCfg, _ := NewAWSConfig(v)
	m := &MockSVC{}
	mv := &MockValidator{}

	ss := &SNSServer{
		Config:               *awsCfg,
		SVC:                  m,
		SNSValidator:         mv,
		channelSize:          channelSize,
		channelClientTimeout: 30 * time.Second,
	}

	r := mux.NewRouter()
	logger := sallust.Default()
	registry, _ := xmetrics.NewRegistry(&xmetrics.Options{}, Metrics)
	awsMetrics := ApplyMetricsData(registry)
	ss.Initialize(r, nil, "", nil, logger, awsMetrics, testNow)

	return ss, m, mv, r
}
