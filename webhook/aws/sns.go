package aws

import (
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
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
	Config           AWSConfig
	subscriptionArn  atomic.Value
	subscriptionData chan string
	SVC              snsiface.SNSAPI
	SelfUrl          *url.URL
	SNSValidator
	logging.Logger
	notificationData chan string
}

// Notifier interface implements the various notification server functionalities
// like Subscribe, Unsubscribe, Publish, NotificationHandler
type Notifier interface {
	Initialize(*mux.Router, *url.URL, http.Handler, logging.Logger, func() time.Time)
	PrepareAndStart()
	Subscribe()
	PublishMessage(string)
	Unsubscribe(string)
	NotificationHandle(http.ResponseWriter, *http.Request) []byte
	ValidateSubscriptionArn(string) bool
}

// NewSNSServer creates SNSServer instance using viper config
func NewSNSServer(v *viper.Viper) (ss *SNSServer, err error) {

	var cfg *AWSConfig
	if cfg, err = NewAWSConfig(v); err != nil {
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
		Config: *cfg,
		SVC:    svc,
	}

	ss.SNSValidator = NewSNSValidator()

	return ss, nil
}

// NewNotifier creates Notifier instance using the viper config
func NewNotifier(v *viper.Viper) (Notifier, error) {
	return NewSNSServer(v)
}

// Initialize initializes the SNSServer fields
// selfURL represents the webhook server URL &url.URL{Scheme:secure,Host:fqdn+port,Path:urlPath}
// handler is the webhook handler to update webhooks @monitor
// SNS POST Notification handler will directly update webhooks list
func (ss *SNSServer) Initialize(rtr *mux.Router, selfUrl *url.URL, handler http.Handler,
	logger logging.Logger, now func() time.Time) {

	if rtr == nil {
		//creating new mux router
		rtr = mux.NewRouter()
	}

	if now == nil {
		now = time.Now
	}

	// Set webhook url path to SNS UrlPath
	// Add unix timestamp to the path to generate unique subArn each time
	var urlPath string
	if strings.HasSuffix(ss.Config.Sns.UrlPath, "/") {
		urlPath = fmt.Sprint(ss.Config.Sns.UrlPath, now().Unix())
	} else {
		urlPath = fmt.Sprint(ss.Config.Sns.UrlPath, "/", now().Unix())
	}

	if selfUrl != nil {
		ss.SelfUrl = selfUrl
		ss.SelfUrl.Path = urlPath
	} else {
		// Test selfurl http://host:port/path
		ss.SelfUrl = &url.URL{
			Scheme: "http",
			Host:   "host:port",
			Path:   urlPath,
		}
	}
	ss.subscriptionData = make(chan string, 5)
	ss.notificationData = make(chan string, 10)

	// set up logger
	if logger != nil {
		ss.Logger = logger
	} else {
		ss.Logger = logging.DefaultLogger()
	}

	ss.Debug("SNS self url endpoint: [%s], protocol [%s]", ss.SelfUrl.String(), ss.SelfUrl.Scheme)

	// Set various SNS POST routes
	ss.SetSNSRoutes(urlPath, rtr, handler)

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
	var quit chan struct{}

	for {
		select {
		case data := <-ss.subscriptionData:
			ss.Debug("listenSubscriptionData ", data)
			ss.subscriptionArn.Store(data)
			if !strings.EqualFold("", data) && !strings.EqualFold("pending confirmation", data) {
				ss.Debug("SNS is ready, subscription arn is cfg %v", data)

				// start listenAndPublishMessage go routine
				quit = make(chan struct{})
				go ss.listenAndPublishMessage(quit)

			} else {
				// stop the listenAndPublishMessage go routine
				// if already running by closing the quit channel
				if nil != quit {
					ss.Error("SNS is not ready now as subscription arn is changed cfg %v", data)
					close(quit)
				}
			}
		}
	}
}

// Validate that SubscriptionArn received in AWS request matches the cached config data
func (ss *SNSServer) ValidateSubscriptionArn(reqSubscriptionArn string) bool {

	if ss.subscriptionArn.Load() == nil {
		ss.Error("SNS subscriptionArn is nil")
		return false
	} else if strings.EqualFold(reqSubscriptionArn, ss.subscriptionArn.Load().(string)) {
		return true
	} else {
		ss.Error(
			"SNS Invalid subscription arn in notification header req %s, cfg %s",
			reqSubscriptionArn, ss.subscriptionArn.Load().(string))
		return false
	}
}
