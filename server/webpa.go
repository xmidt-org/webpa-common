package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/netutil"

	"github.com/Comcast/webpa-common/concurrent"
	"github.com/Comcast/webpa-common/health"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xhttp"
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/justinas/alice"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	DefaultBuild  = "development"
	DefaultServer = "localhost"
	DefaultRegion = "local"
	DefaultFlavor = "development"

	DefaultIdleTimeout       time.Duration = 15 * time.Second
	DefaultReadHeaderTimeout time.Duration = 3 * time.Second
	DefaultWriteTimeout      time.Duration = 3 * time.Second
	DefaultMaxHeaderBytes                  = 8192
)

var (
	// ErrorNoPrimaryAddress is the error returned when no primary address is specified in a WebPA instance
	ErrorNoPrimaryAddress = errors.New("No primary address configured")
)

// executor is an internal type used to start an HTTP server.  *http.Server implements
// this interface.  It can be mocked for testing.
type executor interface {
	Serve(net.Listener) error
	ServeTLS(l net.Listener, certificateFile, keyFile string) error

	ListenAndServe() error
	ListenAndServeTLS(certificateFile, keyFile string) error
}

// Secure exposes the optional certificate information to be used when starting an HTTP server.
type Secure interface {
	// Certificate returns the certificate information associated with this Secure instance.
	// BOTH the returned file paths must be non-empty if a TLS server is desired.
	Certificate() (certificateFile, keyFile string)
}

// Serve is like ListenAndServe, but accepts a custom net.Listener
func Serve(logger log.Logger, s Secure, l net.Listener, e executor) {
	certificateFile, keyFile := s.Certificate()
	if len(certificateFile) > 0 && len(keyFile) > 0 {
		go func() {
			logging.Error(logger).Log(
				logging.ErrorKey(), e.ServeTLS(l, certificateFile, keyFile),
			)
		}()
	} else {
		go func() {
			logging.Error(logger).Log(
				logging.ErrorKey(), e.Serve(l),
			)
		}()
	}
}

// ListenAndServe invokes the appropriate server method based on the secure information.
// If Secure.Certificate() returns both a certificateFile and a keyFile, e.ListenAndServeTLS()
// is called to start the server.  Otherwise, e.ListenAndServe() is used.
func ListenAndServe(logger log.Logger, s Secure, e executor) {
	certificateFile, keyFile := s.Certificate()
	if len(certificateFile) > 0 && len(keyFile) > 0 {
		go func() {
			logging.Error(logger).Log(
				logging.ErrorKey(), e.ListenAndServeTLS(certificateFile, keyFile),
			)
		}()
	} else {
		go func() {
			logging.Error(logger).Log(
				logging.ErrorKey(), e.ListenAndServe(),
			)
		}()
	}
}

type PeerVerifyCallback func([][]byte, [][]*x509.Certificate) error

func DefaultPeerVerifyCallback(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	// Default callback performs no validation 
	return nil
}

// Basic describes a simple HTTP server.  Typically, this struct has its values
// injected via Viper.  See the New function in this package.
type Basic struct {
	Name               string
	Address            string
	CertificateFile    string
	KeyFile            string
	ClientCACertFile   string
	PeerVerifyFunc     PeerVerifyCallback	// Callback func to add peer client cert CN, SAN validation
	LogConnectionState bool

	MaxConnections    int
	DisableKeepAlives bool
	MaxHeaderBytes    int
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
}

func (b *Basic) maxConnections() int {
	if b != nil && b.MaxConnections > 0 {
		return b.MaxConnections
	}

	// no max connections set
	return 0
}

func (b *Basic) SetPeerVerifyCallback( verifyPtr PeerVerifyCallback) {
	b.PeerVerifyFunc = verifyPtr
}

func (b *Basic) maxHeaderBytes() int {
	if b != nil && b.MaxHeaderBytes > 0 {
		return b.MaxHeaderBytes
	}

	return DefaultMaxHeaderBytes
}

func (b *Basic) idleTimeout() time.Duration {
	if b != nil && b.IdleTimeout > 0 {
		return b.IdleTimeout
	}

	return DefaultIdleTimeout
}

func (b *Basic) readHeaderTimeout() time.Duration {
	if b != nil && b.ReadHeaderTimeout > 0 {
		return b.ReadHeaderTimeout
	}

	return DefaultReadHeaderTimeout
}

func (b *Basic) writeTimeout() time.Duration {
	if b != nil && b.WriteTimeout > 0 {
		return b.WriteTimeout
	}

	return DefaultWriteTimeout
}

func (b *Basic) Certificate() (certificateFile, keyFile string) {
	return b.CertificateFile, b.KeyFile
}

// NewListener creates a decorated TCPListener appropriate for this server's configuration.
func (b *Basic) NewListener(logger log.Logger, counter metrics.Counter) (net.Listener, error) {
	l, err := net.Listen("tcp", b.Address)
	if err != nil {
		return nil, err
	}

	l = InstrumentListener(logger, counter, l)
	maxConnections := b.maxConnections()
	if maxConnections > 0 {
		l = netutil.LimitListener(l, maxConnections)
	}

	return l, nil
}

// New creates an http.Server using this instance's configuration.  The given logger is required,
// but the handler may be nil.  If the handler is nil, http.DefaultServeMux is used, which matches
// the behavior of http.Server.
//
// This method returns nil if the configured address is empty, effectively disabling
// this server from startup.
func (b *Basic) New(logger log.Logger, handler http.Handler) *http.Server {
	if len(b.Address) == 0 {
		return nil
	}

	// Adding MTLS support using client CA cert pool
	var tlsConfig *tls.Config
	certificateFile, keyFile := b.Certificate()
	// Only when HTTPS i.e. cert & key present, check for client CA and set TLS config for MTLS
	if len(certificateFile) > 0 && len(keyFile) > 0 && len(b.ClientCACertFile) > 0 {

		caCert, err := ioutil.ReadFile(b.ClientCACertFile)
		if err != nil {
			logging.Error(logger).Log(logging.MessageKey(), "Error in reading ClientCACertFile ",
				logging.ErrorKey(), err)
		} else {
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig = &tls.Config{
				ClientCAs:  caCertPool,
				ClientAuth: tls.RequireAndVerifyClientCert,
			}
			tlsConfig.BuildNameToCertificate()

			// Add verify peer ceritifcate callback for additional validation
			if b.PeerVerifyFunc != nil {
				tlsConfig.VerifyPeerCertificate = b.PeerVerifyFunc
			}
		}
	}

	server := &http.Server{
		Addr:              b.Address,
		Handler:           handler,
		ReadHeaderTimeout: b.readHeaderTimeout(),
		WriteTimeout:      b.writeTimeout(),
		IdleTimeout:       b.idleTimeout(),
		MaxHeaderBytes:    b.maxHeaderBytes(),
		ErrorLog:          NewErrorLog(b.Name, logger),
		TLSConfig:         tlsConfig,
	}

	if b.LogConnectionState {
		server.ConnState = NewConnectionStateLogger(b.Name, logger)
	}

	if b.DisableKeepAlives {
		server.SetKeepAlivesEnabled(false)
	}

	return server
}

// Metric is the configurable factory for a metrics server.
type Metric struct {
	Name               string
	Address            string
	CertificateFile    string
	KeyFile            string
	LogConnectionState bool
	HandlerOptions     promhttp.HandlerOpts
	MetricsOptions     xmetrics.Options
}

func (m *Metric) Certificate() (certificateFile, keyFile string) {
	return m.CertificateFile, m.KeyFile
}

func (m *Metric) NewRegistry(modules ...xmetrics.Module) (xmetrics.Registry, error) {
	// always append the builtin server metrics, which can be overridden in configuration
	modules = append(modules, Metrics)
	return xmetrics.NewRegistry(&m.MetricsOptions, modules...)
}

func (m *Metric) New(logger log.Logger, chain alice.Chain, gatherer stdprometheus.Gatherer) *http.Server {
	if len(m.Address) == 0 {
		return nil
	}

	var (
		mux     = http.NewServeMux()
		handler = chain.Then(promhttp.HandlerFor(gatherer, m.HandlerOptions))
	)

	mux.Handle("/metrics", handler)
	server := &http.Server{
		Addr:              m.Address,
		Handler:           mux,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		WriteTimeout:      DefaultWriteTimeout,
		IdleTimeout:       DefaultIdleTimeout,
		MaxHeaderBytes:    DefaultMaxHeaderBytes,
		ErrorLog:          NewErrorLog(m.Name, logger),
	}

	if m.LogConnectionState {
		server.ConnState = NewConnectionStateLogger(m.Name, logger)
	}

	server.SetKeepAlivesEnabled(false)
	return server
}

// Health represents a configurable factory for a Health server.  As with the Basic type,
// if the Address is not specified, health is considered to be disabled.
//
// Due to a limitation of Viper, this struct does not use an embedded Basic
// instance.  Rather, it duplicates the fields so that Viper can inject them.
type Health struct {
	Name               string
	Address            string
	CertificateFile    string
	KeyFile            string
	LogConnectionState bool
	LogInterval        time.Duration
	Options            []string
}

func (h *Health) Certificate() (certificateFile, keyFile string) {
	return h.CertificateFile, h.KeyFile
}

// NewHealth creates a Health instance from this instance's configuration.  If the Address
// field is not supplied, this method returns nil.
func (h *Health) NewHealth(logger log.Logger, options ...health.Option) *health.Health {
	if len(h.Address) == 0 {
		return nil
	}

	for _, value := range h.Options {
		options = append(options, health.Stat(value))
	}

	return health.New(
		h.LogInterval,
		logger,
		options...,
	)
}

// New creates an HTTP server instance for serving health statistics.  If the health parameter
// is nil, then h.NewHealth is used to create a Health instance.  Otherwise, the health parameter
// is returned as is.
//
// If the Address option is not supplied, the health module is considered to be disabled.  In that
// case, this method simply returns the health parameter as the monitor and a nil server instance.
func (h *Health) New(logger log.Logger, chain alice.Chain, health *health.Health) (*health.Health, *http.Server) {
	if len(h.Address) == 0 {
		// health is disabled
		return nil, nil
	}

	if health == nil {
		if health = h.NewHealth(logger); health == nil {
			// should never hit this case, since NewHealth performs the same
			// Address field check as this method.  but, just to be safe ...
			return nil, nil
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/health", chain.Then(health))

	server := &http.Server{
		Addr:              h.Address,
		Handler:           mux,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		WriteTimeout:      DefaultWriteTimeout,
		IdleTimeout:       DefaultIdleTimeout,
		MaxHeaderBytes:    DefaultMaxHeaderBytes,
		ErrorLog:          NewErrorLog(h.Name, logger),
	}

	if h.LogConnectionState {
		server.ConnState = NewConnectionStateLogger(h.Name, logger)
	}

	server.SetKeepAlivesEnabled(false)
	return health, server
}

// WebPA represents a server component within the WebPA cluster.  It is used for both
// primary servers (e.g. petasos) and supporting, embedded servers such as pprof.
type WebPA struct {
	// ApplicationName is the short identifier for the enclosing application, e.g. "talaria".
	// This value is defaulted to what's passed in via Initialize, but can be changed via injection.
	ApplicationName string

	// Primary is the main server for this application, e.g. petasos.
	Primary Basic

	// Alternate is an alternate server which serves the primary application logic.
	// Used to have the same API served on more than one port and possibly more than
	// one protocol, e.g. HTTP and HTTPS.
	Alternate Basic

	// Health describes the health server for this application.  Note that if the Address
	// is empty, no health server is started.
	Health Health

	// Pprof describes the pprof server for this application.  Note that if the Address
	// is empty, no pprof server is started.
	Pprof Basic

	// Metric describes the metrics provider server for this application
	Metric Metric

	// Build is the build string for the current codebase
	Build string

	// Server is the fully-qualified domain name of this server, typically injected as a fact
	Server string

	// Region is the region in which this server is running, typically injected as a fact
	Region string

	// Flavor is the flavor of this server, typically injected as a fact
	Flavor string

	// Log is the logging configuration for this application.
	Log *logging.Options

	// PeerVerify is the option to be set to verify peer client certificate and details
	PeerVerify bool
}

// build returns the injected build string if available, DefaultBuild otherwise
func (w *WebPA) build() string {
	if w != nil && len(w.Build) > 0 {
		return w.Build
	}

	return DefaultBuild
}

// server returns the injected fully-qualified domain name if available, DefaultServer otherwise
func (w *WebPA) server() string {
	if w != nil && len(w.Server) > 0 {
		return w.Server
	}

	return DefaultServer
}

// region returns the region in which this server is running, or DefaultRegion otherwise
func (w *WebPA) region() string {
	if w != nil && len(w.Region) > 0 {
		return w.Region
	}

	return DefaultRegion
}

// flavor returns the region in which this server is running, or DefaultRegion otherwise
func (w *WebPA) flavor() string {
	if w != nil && len(w.Flavor) > 0 {
		return w.Flavor
	}

	return DefaultFlavor
}

// Prepare gets a WebPA server ready for execution.  This method does not return errors, but the returned
// Runnable may return an error.  The supplied logger will usually come from the New function, but the
// WebPA.Log object can be used to create a different logger if desired.
//
// The caller may pass an arbitrary Health instance.  If this parameter is nil, this method will attempt to
// create one using Health.NewHealth.  In either case, if Health.Address is not supplied, no health server
// will be instantiated.
//
// The caller may also pass a gatherer type. If it is not provided, the default provided by prometheus is used.
//
// The supplied http.Handler is used for the primary server.  If the alternate server has an address,
// it will also be used for that server.  The health server uses an internally create handler, while pprof and metrics
// servers use http.DefaultServeMux.  The health Monitor created from configuration is returned so that other
// infrastructure can make use of it.
func (w *WebPA) Prepare(logger log.Logger, health *health.Health, registry xmetrics.Registry, primaryHandler http.Handler) (health.Monitor, concurrent.Runnable) {
	// allow the health instance to be non-nil, in which case it will be used in favor of
	// the WebPA-configured instance.
	var (
		staticHeaders = xhttp.StaticHeaders(http.Header{
			fmt.Sprintf("X-%s-Build", w.ApplicationName):      {w.build()},
			fmt.Sprintf("X-%s-Server", w.ApplicationName):     {w.server()},
			fmt.Sprintf("X-%s-Region", w.ApplicationName):     {w.region()},
			fmt.Sprintf("X-%s-Flavor", w.ApplicationName):     {w.flavor()},
			fmt.Sprintf("X-%s-Start-Time", w.ApplicationName): {time.Now().UTC().Format(time.RFC822)},
		})

		activeConnections = registry.NewCounter("active_connections")

		healthHandler, healthServer = w.Health.New(logger, alice.New(staticHeaders), health)
		infoLog                     = logging.Info(logger)
	)

	return healthHandler, concurrent.RunnableFunc(func(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
		if healthHandler != nil && healthServer != nil {
			infoLog.Log(logging.MessageKey(), "starting server", "name", w.Health.Name, "address", w.Health.Address)
			ListenAndServe(logger, &w.Health, healthServer)
			healthHandler.Run(waitGroup, shutdown)
		}

		if pprofServer := w.Pprof.New(logger, nil); pprofServer != nil {
			infoLog.Log(logging.MessageKey(), "starting server", "name", w.Pprof.Name, "address", w.Pprof.Address)
			ListenAndServe(logger, &w.Pprof, pprofServer)
		}

		primaryHandler = staticHeaders(w.decorateWithBasicMetrics(registry, primaryHandler))
		if primaryServer := w.Primary.New(logger, primaryHandler); primaryServer != nil {
			listener, err := w.Primary.NewListener(logger, activeConnections.With("primary"))
			if err != nil {
				// TODO: handle cleanup
				return err
			}

			infoLog.Log(logging.MessageKey(), "starting server", "name", w.Primary.Name, "address", w.Primary.Address)
			Serve(logger, &w.Primary, listener, primaryServer)
		} else {
			return ErrorNoPrimaryAddress
		}

		if alternateServer := w.Alternate.New(logger, primaryHandler); alternateServer != nil {
			listener, err := w.Primary.NewListener(logger, activeConnections.With("alternate"))
			if err != nil {
				// TODO: handle cleanup
				return err
			}

			infoLog.Log(logging.MessageKey(), "starting server", "name", w.Alternate.Name, "address", w.Alternate.Address)
			Serve(logger, &w.Alternate, listener, alternateServer)
		}

		if metricsServer := w.Metric.New(logger, alice.New(staticHeaders), registry); metricsServer != nil {
			infoLog.Log(logging.MessageKey(), "starting server", "name", w.Metric.Name, "address", w.Metric.Address)
			ListenAndServe(logger, &w.Metric, metricsServer)
		}

		return nil
	})
}

//decorateWithBasicMetrics wraps a WebPA server handler with basic instrumentation metrics
func (w *WebPA) decorateWithBasicMetrics(p xmetrics.PrometheusProvider, next http.Handler) http.Handler {
	var (
		requestCounterVec    = p.NewCounterVec("api_requests_total")
		inFlightGauge        = p.NewGaugeVec("in_flight_requests").WithLabelValues()
		requestDurationVec   = p.NewHistogramVec("request_duration_seconds")
		requestSizeVec       = p.NewHistogramVec("request_size_bytes")
		responseSizeVec      = p.NewHistogramVec("response_size_bytes")
		timeToWriteHeaderVec = p.NewHistogramVec("time_writing_header_seconds")
	)

	//todo: Example documentation does something interesting with /pull vs. /push endpoints
	//https://godoc.org/github.com/prometheus/client_golang/prometheus/promhttp#InstrumentHandlerDuration
	//for now, let's keep it simple so /metrics only

	return promhttp.InstrumentHandlerInFlight(inFlightGauge,
		promhttp.InstrumentHandlerCounter(requestCounterVec,
			promhttp.InstrumentHandlerDuration(requestDurationVec,
				promhttp.InstrumentHandlerResponseSize(responseSizeVec,
					promhttp.InstrumentHandlerRequestSize(requestSizeVec,
						promhttp.InstrumentHandlerTimeToWriteHeader(timeToWriteHeaderVec, next))),
			),
		),
	)
}
