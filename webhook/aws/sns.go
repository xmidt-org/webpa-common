package aws

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"strings"
	"net/http"
	"net/url"
	"sync/atomic"
	
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/aws/session"
)

type AWSConfig struct {
	AccessKey string    `json:"accessKey"`
	SecretKey string    `json:"secretKey"`
	Env       string    `json:"env"`
	Sns       SNSConfig `json:"sns"`
}

type SNSConfig struct {
	Protocol string `json:"protocol"`
	Region   string `json:"region"`
	TopicArn string `json:"topicArn"`
	UrlPath  string `json:"urlPath"` //uri path to register mux
}

type SNSServer struct {
	Config          	AWSConfig
	subscriptionArn 	atomic.Value
	subscriptionData 	chan string
	SVC             	snsiface.SNSAPI
	SelfUrl         	*url.URL
	Logger				logging.Logger
	notificationData	chan string
}

func NewSNSServer(v *viper.Viper) (ss *SNSServer, err error) {
	
	var cfg *AWSConfig
	if cfg, err = NewAWSConfig(v.Sub(AWSKey)); err != nil {
		return nil, err
	}
	
	cred := credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, "")
	
	sess, aws_err := session.NewSession(&aws.Config{
                Region:      aws.String(cfg.Sns.Region),
                Credentials: cred,
        })
	if aws_err != nil {
		return nil, aws_err
	}

	svc := sns.New(sess)
	// Initialize the server
	ss = &SNSServer{
		Config:   *cfg,
		SVC:      svc,
	}
	
	return ss, nil
}

// NewSNSServer initializes the SNSServer
// selfURL represents the webhook server URL &url.URL{Scheme:secure,Host:fqdn+port,Path:urlPath}
// handler is the webhook handler to update webhooks @monitor 
// SNS POST Notification handler will directly update webhooks list

func (ss *SNSServer) Initialize (rtr *mux.Router, selfUrl *url.URL, handler http.Handler, 
	logger logging.Logger) {
	
	if rtr == nil {
		//creating new mux router
		rtr = mux.NewRouter()
	}

	// Set webhook url path to SNS UrlPath
	if selfUrl != nil {
		selfUrl.Path = ss.Config.Sns.UrlPath
		ss.SelfUrl =  selfUrl
	}
	ss.subscriptionData =  make(chan string, 5)
	ss.notificationData =  make(chan string, 10)
	
	// set up logger
	if logger != nil {
		ss.Logger = logger
	}
	
	ss.logger().Debug("SNS self url endpoint: [%s], protocol [%s]", ss.SelfUrl.String(), ss.SelfUrl.Scheme)

	// Set various SNS POST routes
	ss.SetSNSRoutes(ss.Config.Sns.UrlPath, rtr, handler)
	
}

func (ss *SNSServer) logger() logging.Logger {
	if ss != nil && ss.Logger != nil {
		return ss.Logger
	}
	return logging.DefaultLogger()
}

// Prepare the SNSServer to receive Notifications 
// This better be called after the endpoint http server is started 
// and ready to receive AWS SNS POST messages
// subscribe to the SNS topic
func (ss *SNSServer) PrepareAndStart() {
	
	go ss.listenSubscriptionData()
	
	ss.Subscribe()
}

// Go routine that continuosly listens on SubscriptionData channel for updates to SubscriptionArn
// And stores it to thread safe atomic.Value
// Also checks if its value is NOT "" or "pending confirmation" => SNS Ready
// If SNS ready => ready to receive notification thus starts the listenAndPublishMessage go routine
// to receive notification messages and publish it
// If SNS not ready => stops the listenAndPublishMessage go routine by closing channel
func (ss *SNSServer) listenSubscriptionData() {
	var quit chan int
	
	for {
		select {
			case data := <- ss.subscriptionData:
			ss.logger().Debug("listenSubscriptionData ",data)
			ss.subscriptionArn.Store(data)
			if !strings.EqualFold("", data) && !strings.EqualFold("pending confirmation", data) {
				ss.logger().Debug("SNS is ready, subscription arn is cfg %v", data)
				
				// start listenAndPublishMessage go routine
				quit = make(chan int)
				go ss.listenAndPublishMessage(quit)
				
			} else {
				// stop the listenAndPublishMessage go routine 
				// if already running by closing the quit channel
				if nil != quit {
					ss.logger().Error("SNS is not ready now as subscription arn is changed cfg %v", data)
					close(quit)
				}
			}
		}
	}
}

// Validate that SubscriptionArn received in AWS request matches the cached config data
func (ss *SNSServer) ValidateSubscriptionArn(reqSubscriptionArn string) bool {
	
	if strings.EqualFold(reqSubscriptionArn, ss.subscriptionArn.Load().(string)) {	
		return true
	} else {
		ss.logger().Error(
		"SNS Invalid subscription arn in notification header req %s, cfg %s", 
		reqSubscriptionArn, ss.subscriptionArn.Load().(string))
		 return false
	}
}
