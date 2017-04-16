package aws

import (
	"github.com/gorilla/mux"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
)


const MSG_ATTR  = "scytale.env"

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
	Type             string
	MessageId        string
	Token            string
	TopicArn         string
	Message          string
	SubscribeURL     string
	Timestamp        string
	SignatureVersion string
	Signature        string
	SigningCertURL   string
	UnsubscribeURL   string
	MessageAttributes map[string]MsgAttr
}

// Define handlers for various AWS SNS POST calls
func (ss *SNSServer) SetSNSRoutes(urlPath string, r *mux.Router, handler http.Handler) {
	
	r.HandleFunc(urlPath, ss.SubscribeConfirmHandle).Methods("POST").Headers("x-amz-sns-message-type", "SubscriptionConfirmation")
	r.Handle(urlPath, handler).Methods("POST").Headers("x-amz-sns-message-type", "Notification")
	r.HandleFunc(urlPath, ss.UnsubscribeConfirmHandle).Methods("POST").Headers("x-amz-sns-message-type", "UnsubscribeConfirmation")

}

// Subscribe to AWS SNS Topic to receive notifications
func (ss *SNSServer) Subscribe() bool {
	log.Debug("SNS Subscribe called.")

	params := &sns.SubscribeInput{
		Protocol: aws.String(ss.SelfUrl.Scheme), // Required
		TopicArn: aws.String(ss.Config.Sns.TopicArn), // Required
		Endpoint: aws.String(ss.SelfUrl.String()),
	}
	log.Debug("SNS subscribe params: %#v", params)
	resp, err := ss.SVC.Subscribe(params)

	if err != nil {
		log.Error("SNS subscribe error: %v", err)
		return false
	}

	log.Debug("SNS subscribe resp: %v", resp)
	ss.SubscriptionArn = *resp.SubscriptionArn
	
	return true
}

// POST handler to receive SNS Confirmation Message
func (ss *SNSServer) SubscribeConfirmHandle(rw http.ResponseWriter, req *http.Request) {
	log.Debug("SNS SubscribeConfirmHandle called.")
	defer func() {
		if r := recover(); r != nil {
			log.Error( "SNS SubscribeConfirmHandle recover error %v", r )
		}
	}()
	msg := new(SNSMessage)

	raw, err := DecodeJsonPayload(req, msg)
	if err != nil {
		log.Error("SNS read req body error %v", err)
		return
	}
	//health.SendEvent(HTH.Set("TotalDataPayloadReceived", int(len(raw)) ))

	log.Debug("SNS confirmation payload raw [%v]", string(raw))
	log.Debug("SNS confirmation payload msg [%#v]", msg)

	params := &sns.ConfirmSubscriptionInput{
		Token:    aws.String(msg.Token),    // Required
		TopicArn: aws.String(msg.TopicArn), // Required
	}
	resp, err := ss.SVC.ConfirmSubscription(params)

	if err != nil {
		log.Error("SNS confirm error %v", err)
		return
	}

	log.Debug("SNS confirm response: %v", resp)
	if strings.EqualFold("pending confirmation", *resp.SubscriptionArn) {
		log.Error("SNS pending confirmation")
		ss.SnsReady <- false
		return
	}
	ss.SubscriptionArn = *resp.SubscriptionArn

	ResponseJson(`{"message":"ok"}`, rw)

	log.Trace("SNS SubscribeConfirmHandle req close %v ", req.Close)
}

// Decodes SNS Notification message and returns the actual message which is json webhook content
func (ss *SNSServer) NotificationHandle(rw http.ResponseWriter, req *http.Request) (message string) {
	log.Debug("SNS NotificationHandle called.")
	defer func() {
		if r := recover(); r != nil {
			log.Error( "SNS NotificationHandle recover error %v", r )
		}
	}()
	subArn := req.Header.Get("X-Amz-Sns-Subscription-Arn")
	if subArn != ss.SubscriptionArn {
		log.Error("SNS Invalid subscription arn in notification header req %v", subArn)
		log.Error("SNS Invalid subscription arn in notification header cfg %v", ss.SubscriptionArn)
		ResponseJsonErr(rw, "SubscriptionARN not match", http.StatusBadRequest)
		return
	}
	msg := new(SNSMessage)

	raw, err := DecodeJsonPayload(req, msg)
	if err != nil {
		log.Error("SNS read req body error %v", err)
		ResponseJsonErr(rw, "request body error", http.StatusBadRequest)
		return
	}
	//health.SendEvent(HTH.Set("TotalDataPayloadReceived", int(len(raw)) ))
	
	log.Debug("SNS notification payload raw [%v]", string(raw))
	log.Debug("SNS notification payload msg [%#v]", msg)

	EnvAttr := msg.MessageAttributes[MSG_ATTR]
	log.Trace("SQS notification EnvAttr %v", EnvAttr)

	msgEnv := EnvAttr.Value
	log.Trace("SQS notification msgEnv %v",  msgEnv)
	if msgEnv != ss.Config.Env {
		log.Warn("SQS msg env %v does not match config env %v", msgEnv, ss.Config.Env)
		ResponseJsonErr(rw, "request body error", http.StatusNotAcceptable)
		return
	}
	
	ResponseJson(`{"message":"ok"}`, rw)
	log.Trace("SNS NotificationHandle req close %v", req.Close)
	
	return msg.Message
}

func (ss *SNSServer) PublishMessage(message string) {
	log.Debug("SNS PublishMessage called %v ", message)
	defer func() {
		if r := recover(); r != nil {
			log.Error( "SNS PublishMessage recover error %v", r )
		}
	}()

	params := &sns.PublishInput{
		Message: aws.String(message), // Required
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			MSG_ATTR: { // Required
				DataType:    aws.String("String"), // Required
				StringValue: aws.String(ss.Config.Env),
			},
		},
		MessageStructure: aws.String("messageStructure"),
		Subject:          aws.String("subject new"),
		TopicArn:         aws.String(ss.Config.Sns.TopicArn),
	}
	resp, err := ss.SVC.Publish(params)

	if err != nil {
		log.Error("SNS send message error %v", err)

		return
	}
	log.Debug("SNS send message resp: %v", resp)
	//health.SendEvent(HTH.Set("TotalDataPayloadSent", int(len([]byte(resp.GoString()))) ))

}

func (ss *SNSServer) UnsubscribeConfirmHandle(rw http.ResponseWriter, req *http.Request) {
	log.Debug("SNS UnubscribeConfirmHandle called.")
	
	// TODO
}
