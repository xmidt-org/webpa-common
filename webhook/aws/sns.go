package aws

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/gorilla/mux"
	"strings"
	"net/http"
	"net/url"
	"fmt"
	"sync"
	
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sns"
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
	Config          *AWSConfig
	SubscriptionArn string
	SVC             *sns.SNS
	SelfUrl         url.URL
	Logger			logging.Logger
	// mutex used to protect SubscriptionArn read & write
	sync.RWMutex
}

// NewSNSServer initializes the SNSServer
// selfURL represents the webhook server URL &url.URL{Scheme:secure,Host:fqdn+port,Path:urlPath}
// handler is the webhook handler to update webhooks @monitor 
// SNS POST Notification handler will directly update webhooks list

func NewSNSServer(cfg *AWSConfig, logger logging.Logger, rtr *mux.Router, 
	selfUrl url.URL, handler http.Handler) (ss *SNSServer, err error) {
	
	if cfg == nil {
		return nil, fmt.Errorf("Invalid AWS Config")
	}
	
	if rtr == nil {
		//creating new mux router
		rtr = mux.NewRouter()
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

	// Set webhook url path to SNS UrlPath
	selfUrl.Path = cfg.Sns.UrlPath

	// Initialize the server
	ss = &SNSServer{
		Config:   cfg,
		SVC:      svc,
		SelfUrl:  selfUrl,
	}
	
	// set up logger
	ss.Logger = logger
	
	ss.logger().Debug("SNS self url endpoint: [%s], protocol [%s]", ss.SelfUrl.String(), ss.SelfUrl.Scheme)

	// Set various SNS POST routes
	ss.SetSNSRoutes(cfg.Sns.UrlPath, rtr, handler)
	
	return ss, nil
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
// subscribe to the SNS topic, wait for snsReady
// validate the confirmation SubscriptionArn
func (ss *SNSServer) PrepareAndStart() {
	ss.Subscribe()
}

// Returns true if the SNS is ready to accept notifications
// Synchronized using read lock
func (ss *SNSServer) IsReady() bool {
	var ready bool
	ss.RLock()
	if !strings.EqualFold("pending confirmation", "") && !strings.EqualFold("pending confirmation", ss.SubscriptionArn) {	
		ready = true
	} else {
		ss.logger().Error("SNS is not yet ready, subscription arn in cfg %v", 
			ss.SubscriptionArn)
		ready = false
	}	
	ss.RUnlock()
	return ready
}

// Validate that SubscriptionArn received in AWS request matches the cached config data
func (ss *SNSServer) ValidateSubscriptionArn(reqSubscriptionArn string) bool {
	var valid bool
	ss.RLock()
	if strings.EqualFold(reqSubscriptionArn, ss.SubscriptionArn) {	
		valid = true
	} else {
		ss.logger().Error(
		"SNS Invalid subscription arn in notification header req %s, cfg %s", 
		reqSubscriptionArn, ss.SubscriptionArn)
		valid = false
	}
	ss.RUnlock()
	return valid
}


// Synchronized block of code to update SubscriptionArn
// Write Thread synchronization using lock
func (ss *SNSServer) UpdateSubscriptionArn(respSubscriptionArn *string) {
	ss.Lock()
	ss.SubscriptionArn = *respSubscriptionArn
	ss.Unlock()
}
