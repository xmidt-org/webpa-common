package aws

import (
	"container/list"
	"fmt"
	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/logging"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MSG_ATTR           = "scytale.env"
	SNS_VALIDATION_ERR = "SNS signature validation error"
)

/* http://docs.aws.amazon.com/sns/latest/dg/SendMessageToHttp.html
POST / HTTP/1.1
x-amz-sns-message-type: SubscriptionConfirmation
x-amz-sns-message-id: 165545c9-2a5c-472c-8df2-7ff2be2b3b1b
x-amz-sns-topic-arn: arn:aws:sns:us-west-2:123456789012:MyTopic
Content-Length: 1336
Content-Type: text/plain; charset=UTF-8
Host: example.com
Connection: Keep-Alive
User-Agent: Amazon Simple Notification Service Agent

{
  "Type" : "SubscriptionConfirmation",
  "MessageId" : "165545c9-2a5c-472c-8df2-7ff2be2b3b1b",
  "Token" : "2336412f37fb687f5d51e6e241d09c805a5a57b30d712f794cc5f6a988666d92768dd60a747ba6f3beb71854e285d6ad02428b09ceece29417f1f02d609c582afbacc99c583a916b9981dd2728f4ae6fdb82efd087cc3b7849e05798d2d2785c03b0879594eeac82c01f235d0e717736",
  "TopicArn" : "arn:aws:sns:us-west-2:123456789012:MyTopic",
  "Message" : "You have chosen to subscribe to the topic arn:aws:sns:us-west-2:123456789012:MyTopic.\nTo confirm the subscription, visit the SubscribeURL included in this message.",
  "SubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn:aws:sns:us-west-2:123456789012:MyTopic&Token=2336412f37fb687f5d51e6e241d09c805a5a57b30d712f794cc5f6a988666d92768dd60a747ba6f3beb71854e285d6ad02428b09ceece29417f1f02d609c582afbacc99c583a916b9981dd2728f4ae6fdb82efd087cc3b7849e05798d2d2785c03b0879594eeac82c01f235d0e717736",
  "Timestamp" : "2012-04-26T20:45:04.751Z",
  "SignatureVersion" : "1",
  "Signature" : "EXAMPLEpH+DcEwjAPg8O9mY8dReBSwksfg2S7WKQcikcNKWLQjwu6A4VbeS0QHVCkhRS7fUQvi2egU3N858fiTDN6bkkOxYDVrY0Ad8L10Hs3zH81mtnPk5uvvolIC1CXGu43obcgFxeL3khZl8IKvO61GWB6jI9b5+gLPoBc1Q=",
  "SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem"
  }

POST / HTTP/1.1
x-amz-sns-message-type: Notification
x-amz-sns-message-id: 22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324
x-amz-sns-topic-arn: arn:aws:sns:us-west-2:123456789012:MyTopic
x-amz-sns-subscription-arn: arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96
Content-Length: 773
Content-Type: text/plain; charset=UTF-8
Host: example.com
Connection: Keep-Alive
User-Agent: Amazon Simple Notification Service Agent

{
  "Type" : "Notification",
  "MessageId" : "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
  "TopicArn" : "arn:aws:sns:us-west-2:123456789012:MyTopic",
  "Subject" : "My First Message",
  "Message" : "Hello world!",
  "Timestamp" : "2012-05-02T00:54:06.655Z",
  "SignatureVersion" : "1",
  "Signature" : "EXAMPLEw6JRNwm1LFQL4ICB0bnXrdB8ClRMTQFGBqwLpGbM78tJ4etTwC5zU7O3tS6tGpey3ejedNdOJ+1fkIp9F2/LmNVKb5aFlYq+9rk9ZiPph5YlLmWsDcyC5T+Sy9/umic5S0UQc2PEtgdpVBahwNOdMW4JPwk0kAJJztnc=",
  "SigningCertURL" : "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
  "UnsubscribeURL" : "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96"
  }
*/
type MsgAttr struct {
	Type  string
	Value string
}
type SNSMessage struct {
	Type              string
	MessageId         string
	Token             string
	TopicArn          string
	Subject           string
	Message           string
	SubscribeURL      string
	Timestamp         string
	SignatureVersion  string
	Signature         string
	SigningCertURL    string
	UnsubscribeURL    string
	MessageAttributes map[string]MsgAttr
}

// Define handlers for various AWS SNS POST calls
func (ss *SNSServer) SetSNSRoutes(urlPath string, r *mux.Router, handler http.Handler) {

	r.HandleFunc(urlPath, ss.SubscribeConfirmHandle).Methods("POST").Headers("x-amz-sns-message-type", "SubscriptionConfirmation")
	if handler != nil {
		ss.debugLog.Log(logging.MessageKey(), "handler not nil", "urlPath", urlPath)
		// handler is supposed to be wrapper that inturn calls NotificationHandle
		r.Handle(urlPath, handler).Methods("POST").Headers("x-amz-sns-message-type", "Notification")
	} else {
		ss.debugLog.Log(logging.MessageKey(), "handler nil", "urlPath", urlPath)
		// if no wrapper handler available then define anonymous handler and directly call NotificationHandle
		r.HandleFunc(urlPath, func(rw http.ResponseWriter, req *http.Request) {
			ss.NotificationHandle(rw, req)
		}).Methods("POST").Headers("x-amz-sns-message-type", "Notification")
	}
}

// Subscribe to AWS SNS Topic to receive notifications
func (ss *SNSServer) Subscribe() {

	params := &sns.SubscribeInput{
		Protocol: aws.String(ss.SelfUrl.Scheme),      // Required
		TopicArn: aws.String(ss.Config.Sns.TopicArn), // Required
		Endpoint: aws.String(ss.SelfUrl.String()),
	}

	ss.debugLog.Log("subscribeParams", params)

	resp, err := ss.SVC.Subscribe(params)
	if err != nil {
		attemptNum := 1
		ss.errorLog.Log(logging.MessageKey(), "SNS subscribe error", "attempt", attemptNum, logging.ErrorKey(), err)
		attemptNum++

		// this is so tests do not timeout
		if *params.TopicArn == "arn:aws:sns:us-east-1:1234:test-topic" ||
			*params.Endpoint == "http://host:port/api/v2/aws/sns" {
			return
		}

		for {
			time.Sleep(time.Second * 5)

			resp, err = ss.SVC.Subscribe(params)
			if err != nil {
				ss.errorLog.Log(logging.MessageKey(), "SNS subscribe error", "attempt", attemptNum, logging.ErrorKey(), err)
			} else {
				break
			}

			attemptNum++
		}
	}

	ss.debugLog.Log("subscribeResponse", resp)

	// Add SubscriptionArn to subscription data channel
	ss.subscriptionData <- *resp.SubscriptionArn
}

// POST handler to receive SNS Confirmation Message
func (ss *SNSServer) SubscribeConfirmHandle(rw http.ResponseWriter, req *http.Request) {

	msg := new(SNSMessage)

	raw, err := DecodeJSONMessage(req, msg)
	if err != nil {
		ss.errorLog.Log(logging.MessageKey(), "SNS read req body error", logging.ErrorKey(), err)
		httperror.Format(rw, http.StatusBadRequest, "request body error")
		return
	}

	// Verify SNS Message authenticity by verifying signature
	valid, v_err := ss.Validate(msg)
	if !valid || v_err != nil {
		ss.errorLog.Log(logging.MessageKey(), "SNS signature validation error", logging.ErrorKey(), v_err)
		httperror.Format(rw, http.StatusBadRequest, SNS_VALIDATION_ERR)
		return
	}

	// Validate that SubscriptionConfirmation is for the topic you desire to subscribe to
	if !strings.EqualFold(msg.TopicArn, ss.Config.Sns.TopicArn) {
		ss.errorLog.Log(
			logging.MessageKey(), "SNS subscription confirmation TopicArn mismatch",
			"received", msg.TopicArn,
			"expected", ss.Config.Sns.TopicArn)
		httperror.Format(rw, http.StatusBadRequest, "TopicArn does not match")
		return
	}

	// TODO: health.SendEvent(HTH.Set("TotalDataPayloadReceived", int(len(raw)) ))

	ss.debugLog.Log(
		logging.MessageKey(), "SNS confirmation payload",
		"raw", string(raw),
		"msg", msg,
	)

	params := &sns.ConfirmSubscriptionInput{
		Token:    aws.String(msg.Token),    // Required
		TopicArn: aws.String(msg.TopicArn), // Required
	}
	resp, err := ss.SVC.ConfirmSubscription(params)
	if err != nil {
		ss.errorLog.Log(logging.MessageKey(), "SNS confirm error", logging.ErrorKey(), err)
		// TODO return error response
		return
	}

	ss.debugLog.Log(logging.MessageKey(), "SNS confirm response", "response", resp)

	// Add SubscriptionArn to subscription data channel
	ss.subscriptionData <- *resp.SubscriptionArn

}

// Decodes SNS Notification message and returns
// the actual message which is json webhook content
func (ss *SNSServer) NotificationHandle(rw http.ResponseWriter, req *http.Request) []byte {

	subArn := req.Header.Get("X-Amz-Sns-Subscription-Arn")
	if !ss.ValidateSubscriptionArn(subArn) {
		// Returning HTTP 500 error such that AWS will retry and meanwhile subscriptionConfirmation will be received
		httperror.Format(rw, http.StatusInternalServerError, "SubscriptionARN does not match")
		return nil
	}

	msg := new(SNSMessage)

	raw, err := DecodeJSONMessage(req, msg)
	if err != nil {
		ss.errorLog.Log(logging.MessageKey(), "SNS read req body error", logging.ErrorKey(), err)
		httperror.Format(rw, http.StatusBadRequest, "request body error")
		return nil
	}

	// Verify SNS Message authenticity by verifying signature
	valid, v_err := ss.Validate(msg)
	if !valid || v_err != nil {
		ss.errorLog.Log(logging.MessageKey(), "SNS signature validation error", logging.ErrorKey(), v_err)
		httperror.Format(rw, http.StatusBadRequest, SNS_VALIDATION_ERR)
		return nil
	}
	// TODO: health.SendEvent(HTH.Set("TotalDataPayloadReceived", int(len(raw)) ))

	ss.debugLog.Log(
		logging.MessageKey(), "SNS notification payload",
		"raw", string(raw),
		"msg", msg,
	)

	// Validate that SubscriptionConfirmation is for the topic you desire to subscribe to
	if !strings.EqualFold(msg.TopicArn, ss.Config.Sns.TopicArn) {
		ss.errorLog.Log(
			logging.MessageKey(), "SNS notification TopicArn mismatch",
			"received", msg.TopicArn,
			"expected", ss.Config.Sns.TopicArn)
		httperror.Format(rw, http.StatusBadRequest, "TopicArn does not match")
		return nil
	}

	EnvAttr := msg.MessageAttributes[MSG_ATTR]
	msgEnv := EnvAttr.Value
	ss.debugLog.Log(logging.MessageKey(), "SNS notification", "envAttr", EnvAttr, "msgEnv", msgEnv)
	if msgEnv != ss.Config.Env {
		ss.errorLog.Log(logging.MessageKey(), "SNS environment mismatch", "msgEnv", msgEnv, "config", ss.Config.Env)
		httperror.Format(rw, http.StatusBadRequest, "SNS Msg config env does not match")
		return nil
	}

	return []byte(msg.Message)
}

// Publish Notification message to AWS SNS topic
func (ss *SNSServer) PublishMessage(message string) {

	ss.debugLog.Log(logging.MessageKey(), "SNS PublishMessage", "called", message)

	// push Notification message onto notif data channel
	ss.notificationData <- message
}

// listenAndPublishMessage go routine listens for data on notificationData channel
// NS publishes it to SNS
// This go Routine is started when SNS Ready and stopped when SNS is not Ready
func (ss *SNSServer) listenAndPublishMessage(quit <-chan struct{}) {
	for {
		select {
		case message := <-ss.notificationData:

			params := &sns.PublishInput{
				Message: aws.String(message), // Required
				MessageAttributes: map[string]*sns.MessageAttributeValue{
					MSG_ATTR: { // Required
						DataType:    aws.String("String"), // Required
						StringValue: aws.String(ss.Config.Env),
					},
				},
				Subject:  aws.String("new webhook"),
				TopicArn: aws.String(ss.Config.Sns.TopicArn),
			}
			resp, err := ss.SVC.Publish(params)

			if err != nil {
				ss.errorLog.Log(logging.MessageKey(), "SNS send message error", logging.ErrorKey(), err)
			}
			ss.debugLog.Log(logging.MessageKey(), "SNS send message", "response", resp)
		// TODO : health.SendEvent(HTH.Set("TotalDataPayloadSent", int(len([]byte(resp.GoString()))) ))

		// To terminate the go routine when SNS is not ready, so dont allow publish message
		case <-quit:
			return
		}
	}
}

// Unsubscribe from receiving notifications
func (ss *SNSServer) Unsubscribe(subArn string) {
	var subscriptionArn string
	if !strings.EqualFold(subArn, "") {
		subscriptionArn = subArn
	} else {
		subscriptionArn = ss.subscriptionArn.Load().(string)
	}
	params := &sns.UnsubscribeInput{
		SubscriptionArn: aws.String(subscriptionArn), // Required
	}

	resp, err := ss.SVC.Unsubscribe(params)

	if err != nil {
		ss.errorLog.Log(logging.MessageKey(), "SNS Unsubscribe error", logging.ErrorKey(), err)
	}

	ss.debugLog.Log(logging.MessageKey(), "Successfully unsubscribed from the SNS topic", "response", resp)
}

func (ss *SNSServer) UnsubscribeOldSubscriptions() {
	unsubList, err := ss.ListSubscriptionsByMatchingEndpoint()
	if err != nil || unsubList == nil {
		return
	}

	subArn := unsubList.Front()
	for subArn != nil {
		ss.Unsubscribe(subArn.Value.(string))
		subArn = subArn.Next()
	}

}

func (ss *SNSServer) ListSubscriptionsByMatchingEndpoint() (*list.List, error) {
	var unsubscribeList *list.List
	var next bool
	next = true

	// Extract current timestamp and endpoint
	var timestamp, currentTimestamp int64
	var err error
	timeStr := strings.TrimPrefix(ss.SelfUrl.Path, ss.Config.Sns.UrlPath)
	timeStr = strings.TrimPrefix(timeStr, "/")
	currentTimestamp, err = strconv.ParseInt(timeStr, 10, 64)
	if nil != err {
		ss.errorLog.Log(logging.MessageKey(), "SNS List Subscriptions timestamp parse error", logging.ErrorKey(), err)
		return nil, err
	}
	endpoint := strings.TrimSuffix(ss.SelfUrl.String(), timeStr)
	endpoint = strings.TrimSuffix(endpoint, "/")
	ss.debugLog.Log("currentEndpoint", endpoint, "timestamp", currentTimestamp)

	params := &sns.ListSubscriptionsByTopicInput{
		TopicArn: aws.String(ss.Config.Sns.TopicArn),
	}

	for next == true {

		resp, err := ss.SVC.ListSubscriptionsByTopic(params)
		if nil != err {
			ss.errorLog.Log(logging.MessageKey(), "SNS ListSubscriptionsByTopic error", logging.ErrorKey(), err)
			return nil, err
		}

		for _, sub := range resp.Subscriptions {
			timestamp = 0
			if !strings.EqualFold(*sub.Endpoint, ss.SelfUrl.String()) &&
				strings.Contains(*sub.Endpoint, endpoint) {

				fmt.Sscanf(*sub.Endpoint, endpoint+"/%d", &timestamp)
				if timestamp == 0 || timestamp < currentTimestamp {

					if unsubscribeList == nil {
						unsubscribeList = list.New()
						unsubscribeList.PushBack(*sub.SubscriptionArn)
					} else {
						unsubscribeList.PushBack(*sub.SubscriptionArn)
					}
				}
			}
		}

		if resp.NextToken != nil {
			next = true
			params = params.SetNextToken(*resp.NextToken)
		} else {
			next = false
		}

	}

	if unsubscribeList != nil {
		ss.debugLog.Log(logging.MessageKey(), "SNS Unsubscribe List", "length", unsubscribeList.Len())
	}
	return unsubscribeList, nil
}
