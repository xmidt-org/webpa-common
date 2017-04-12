package aws

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/gorilla/mux"
	"strings"
	"net/http"
	"net/url"
	"fmt"
	
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
	SnsReady		chan bool
}

var log logging.Logger

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
	
	ss.SnsReady = make(chan bool)
	
	// set up logger
	ss.Logger = logger
	log = ss.logger()
	
	log.Debug("SNS self url endpoint: [%s], protocol [%s]", ss.SelfUrl.String(), ss.SelfUrl.Scheme)

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
// This should be called only after the endpoint http server is started and ready to receive AWS SNS POST messages
// subscribe to the SNS topic, wait for snsReady
// validate the confirmation SubscriptionArn
func (ss *SNSServer) Prepare() bool {
	
	ss.Subscribe()
	
	ready := <- ss.SnsReady
	
	if ready == true {
		if strings.EqualFold("pending confirmation", ss.SubscriptionArn) {
			log.Error("SNS is ready but SubscriptionArn is %s", ss.SubscriptionArn)
			return false 
		}
	}
	return true
}

