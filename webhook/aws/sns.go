package aws

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/miekg/dns"
	"github.com/spf13/viper"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xmetrics"

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
	Protocol    string `json:"protocol"`
	Region      string `json:"region"`
	TopicArn    string `json:"topicArn"`
	UrlPath     string `json:"urlPath"` //uri path to register mux
	AwsEndpoint string `json:"awsEndpoint"`
}

type SNSServer struct {
	Config          AWSConfig
	subscriptionArn atomic.Value
	SVC             snsiface.SNSAPI
	SelfUrl         *url.URL
	SOAProvider     string
	SNSValidator
	notificationData     chan string
	channelSize          int64
	channelClientTimeout time.Duration

	errorLog                    log.Logger
	debugLog                    log.Logger
	metrics                     AWSMetrics
	snsNotificationReceivedChan chan int
	waitForDns                  time.Duration
}

// Notifier interface implements the various notification server functionalities
// like Subscribe, Unsubscribe, Publish, NotificationHandler
type Notifier interface {
	Initialize(*mux.Router, *url.URL, string, http.Handler, log.Logger, xmetrics.Registry, func() time.Time)
	PrepareAndStart()
	Subscribe()
	PublishMessage(string) error
	Unsubscribe(string)
	NotificationHandle(http.ResponseWriter, *http.Request) []byte
	ValidateSubscriptionArn(string) bool
	SNSNotificationReceivedCounter(int)
	DnsReady() error
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
		Endpoint:    aws.String(cfg.Sns.AwsEndpoint),
		Credentials: cred,
	})
	if aws_err != nil {
		return nil, aws_err
	}

	svc := sns.New(sess)
	// Initialize the server
	ss = &SNSServer{
		Config:               *cfg,
		SVC:                  svc,
		channelSize:          50,
		channelClientTimeout: 30 * time.Second,
	}

	if v != nil && v.IsSet("waitForDns") {
		ss.waitForDns = v.GetDuration("waitForDns")
	}

	if v != nil && v.IsSet("sns.channelSize") {
		ss.channelSize = v.GetInt64("sns.channelSize")
	}

	if v != nil && v.IsSet("sns.channelClientTimeout") {
		ss.channelClientTimeout = v.GetDuration("sns.channelClientTimeout")
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
func (ss *SNSServer) Initialize(rtr *mux.Router, selfUrl *url.URL, soaProvider string,
	handler http.Handler, logger log.Logger, registry xmetrics.Registry, now func() time.Time) {

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

	ss.SOAProvider = soaProvider

	ss.notificationData = make(chan string, ss.channelSize)

	// set up logger
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	ss.errorLog = logging.Error(logger)
	ss.debugLog = logging.Debug(logger)

	ss.metrics = ApplyMetricsData(registry)
	ss.snsNotificationReceivedChan = ss.SNSNotificationReceivedInit()

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

//DnsReady blocks until the primary server's DNS is up and running or
//until the timeout is reached
//if timeout value is 0s it will try forever
func (ss *SNSServer) DnsReady() (e error) {

	// if an SOA provider isn't given, we're done
	if ss.SOAProvider == "" {
		return nil
	}

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	if ss.waitForDns > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), ss.waitForDns)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	// Creating the dns client for our query
	client := dns.Client{
		Net: "tcp", // tcp to connect to the SOA provider? or udp (default)?
		Dialer: &net.Dialer{
			Timeout: ss.waitForDns,
		},
	}
	// the message contains what we are looking for - the SOA record of the host
	msg := dns.Msg{}
	msg.SetQuestion(strings.SplitN(ss.SelfUrl.Host, ":", 2)[0]+".", dns.TypeANY)

	defer cancel()

	var check = func() <-chan struct{} {
		var channel = make(chan struct{})

		go func(c chan struct{}) {
			var (
				err      error
				response *dns.Msg
			)

			for {
				// sending the dns query to the soa provider
				response, _, err = client.Exchange(&msg, ss.SOAProvider)
				// if we found a record, then we are done
				if err == nil && response != nil && response.Rcode == dns.RcodeSuccess && len(response.Answer) > 0 {
					c <- struct{}{}
					ss.metrics.DnsReady.Add(1.0)
					return
				}
				// otherwise, we keep trying
				ss.metrics.DnsReadyQueryCount.Add(1.0)
				ss.debugLog.Log(logging.MessageKey(), "checking if server's DNS is ready",
					"endpoint", strings.SplitN(ss.SelfUrl.Host, ":", 2)[0]+".", logging.ErrorKey(), err, "response", response)
				time.Sleep(time.Second)
			}
		}(channel)

		return channel
	}

	select {
	case <-check():
	case <-ctx.Done():
		e = ctx.Err()
	}

	return
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

// SNSNotificationReceivedCounter relays response code data to be aggregated in metrics
func (ss *SNSServer) SNSNotificationReceivedCounter(code int) {
	ss.snsNotificationReceivedChan <- code
}

// SNSNotificationReceivedInit initializes metrics counters and returns a channel to send response codes to count
func (ss *SNSServer) SNSNotificationReceivedInit() chan int {
	// notification channel
	notifyChan := make(chan int)

	// create counters
	internalErr := ss.metrics.SNSNotificationReceived.With("code", strconv.Itoa(http.StatusInternalServerError))
	badRequest := ss.metrics.SNSNotificationReceived.With("code", strconv.Itoa(http.StatusBadRequest))
	okay := ss.metrics.SNSNotificationReceived.With("code", strconv.Itoa(http.StatusOK))
	other := ss.metrics.SNSNotificationReceived.With("code", "other")

	// set values to 0
	internalErr.Add(0.0)
	badRequest.Add(0.0)
	okay.Add(0.0)
	other.Add(0.0)

	fn := func() {
		for {
			code := <-notifyChan
			switch code {
			case http.StatusInternalServerError:
				internalErr.Add(1.0)
			case http.StatusBadRequest:
				badRequest.Add(1.0)
			case http.StatusOK:
				okay.Add(1.0)
			default:
				other.Add(1.0)
			}
		}
	}

	go fn()

	return notifyChan
}
