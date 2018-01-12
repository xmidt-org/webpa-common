package aws

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"

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
	Config          AWSConfig
	subscriptionArn atomic.Value
	SVC     snsiface.SNSAPI
	SelfUrl *url.URL
	SNSValidator
	notificationData chan string

	errorLog log.Logger
	debugLog log.Logger
	metrics  Metrics
}

// Notifier interface implements the various notification server functionalities
// like Subscribe, Unsubscribe, Publish, NotificationHandler
type Notifier interface {
	Initialize(*mux.Router, *url.URL, http.Handler, log.Logger, *xmetrics.Registry, func() time.Time)
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
	logger log.Logger, registry *xmetrics.Registry, now func() time.Time) {

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
	ss.notificationData = make(chan string, 10)

	// set up logger
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	ss.errorLog = logging.Error(logger)
	ss.debugLog = logging.Debug(logger)

	if registry != nil {
		ss.metrics = AddMetrics(*registry)
	} else {
		o := &xmetrics.Options{}
		registry, err := xmetrics.NewRegistry(o)
		if err != nil {
			ss.errorLog.Log(logging.MessageKey(), "failed to create default registry", "error", err)
		}
		ss.metrics = AddMetrics(&registry)
	}


	ss.debugLog.Log("selfURL", ss.SelfUrl.String(), "protocol", ss.SelfUrl.Scheme)

	// Set various SNS POST routes
	ss.SetSNSRoutes(urlPath, rtr, handler)

}

// Prepare the SNSServer to receive Notifications
// This better be called after the endpoint http server is started
// and ready to receive AWS SNS POST messages
// subscribe to the SNS topic
func (ss *SNSServer) PrepareAndStart() {

	ss.Subscribe()
}

// Validate that SubscriptionArn received in AWS request matches the cached config data
func (ss *SNSServer) ValidateSubscriptionArn(reqSubscriptionArn string) bool {

	if ss.subscriptionArn.Load() == nil {
		ss.errorLog.Log(logging.MessageKey(), "SNS subscriptionArn is nil")
		return false
	} else if strings.EqualFold(reqSubscriptionArn, ss.subscriptionArn.Load().(string)) {
		return true
	} else {
		ss.errorLog.Log(
			logging.MessageKey(), "SNS Invalid subscription",
			"reqSubscriptionArn", reqSubscriptionArn,
			"cfg", ss.subscriptionArn.Load().(string),
		)
		return false
	}
}

// helper function to report the list size value
func (ss *SNSServer) ReportListSize(size int) {
	ss.metrics.ListSize.Set(float64(size))
}

// helper function to get the right subscription attempts counter to increment
func (ss *SNSServer) SNSSubscriptionAttemptCounter(code int) metrics.Counter {
	if code == -1 {
		return ss.metrics.SNSSubscribeAttempt.With("code", "failure")
	}

	return ss.metrics.SNSSubscribeAttempt.With("code", "okay")
}

// helper function to get the right notification received counter to increment
func (ss *SNSServer) SNSNotificationReceivedCounter(code int) metrics.Counter {
	if code == -1 {
		return ss.metrics.SNSNotificationReceived.With("code", "failure")
	}

	s := strconv.Itoa(code)
	return ss.metrics.SNSNotificationReceived.With("code", s)
}

