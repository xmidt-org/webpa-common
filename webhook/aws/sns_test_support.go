package aws

import (
	"github.com/gorilla/mux"
)

type ErrResp struct {
	Code    int
	Message string
}

const (
	TEST_AWS_CONFIG = `{
	"aws": {
        "accessKey": "test-accessKey",
        "secretKey": "test-secretKey",
        "env": "test",
        "sns" : {
	        "region" : "us-east-1",
            "protocol" : "http",
			"topicArn" : "arn:aws:sns:us-east-1:1234:test-topic", 
			"urlPath" : "/api/v2/aws/sns",
			"awsEndpoint" : "http://example.com"
    } } }`
	TEST_SUB_MSG = `{
	"Type" : "SubscriptionConfirmation",
	"MessageId" : "165545c9-2a5c-472c-8df2-7ff2be2b3b1b",
	"Token" : "2336412f37fb687f5d51e6e241d09c805a5a57b30d712f794cc5f6a988666d92768dd60a747ba6f3beb71854e285d6ad02428b09ceece29417f1f02d609c582afbacc99c583a916b9981dd2728f4ae6fdb82efd087cc3b7849e05798d2d2785c03b0879594eeac82c01f235d0e717736",
	"TopicArn" : "arn:aws:sns:us-east-1:1234:test-topic",
	"Message" : "You have chosen to subscribe to the topic arn:aws:sns:us-east-1:1234:test-topic.\nTo confirm the subscription, visit the SubscribeURL included in this message.",
	"SubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn:aws:sns:us-west-2:123456789012:MyTopic&Token=2336412f37fb687f5d51e6e241d09c805a5a57b30d712f794cc5f6a988666d92768dd60a747ba6f3beb71854e285d6ad02428b09ceece29417f1f02d609c582afbacc99c583a916b9981dd2728f4ae6fdb82efd087cc3b7849e05798d2d2785c03b0879594eeac82c01f235d0e717736",
	"Timestamp" : "2012-04-26T20:45:04.751Z",
	"SignatureVersion" : "1",
	"Signature" : "EXAMPLEpH+DcEwjAPg8O9mY8dReBSwksfg2S7WKQcikcNKWLQjwu6A4VbeS0QHVCkhRS7fUQvi2egU3N858fiTDN6bkkOxYDVrY0Ad8L10Hs3zH81mtnPk5uvvolIC1CXGu43obcgFxeL3khZl8IKvO61GWB6jI9b5+gLPoBc1Q=",
	"SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem"
	}`
	TEST_HOOK = `{
		"config": {
			"url": "http://127.0.0.1:8080/test",
			"content_type": "json",
			"secret": ""
		},
		"matcher": {
			"device_id": [
				".*"
			]
		},
		"events": [
			"transaction-status",
			"SYNC_NOTIFICATION"
		]
	}`
	NOTIFY_HOOK_MSG = `{
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
	} }`
	TEST_NOTIF_ERR_MSG = `{
	"Type" : "Notification",
	"MessageId" : "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
	"TopicArn" : "Invalid-topic",
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
)

func SetUpTestNotifier() (Notifier, *MockSVC, *MockValidator, *mux.Router) {

	awsCfg, _ := NewAWSConfig(nil)
	m := &MockSVC{}
	mv := &MockValidator{}

	ss := &SNSServer{
		Config:       *awsCfg,
		SVC:          m,
		SNSValidator: mv,
	}

	r := mux.NewRouter()

	return ss, m, mv, r
}
