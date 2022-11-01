package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/justinas/alice"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/sallust/sallusthttp"
	"github.com/xmidt-org/webpa-common/v2/concurrent"
	"github.com/xmidt-org/webpa-common/v2/health"
	"github.com/xmidt-org/webpa-common/v2/xhttp"
	"github.com/xmidt-org/webpa-common/v2/xlistener"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
	"go.uber.org/zap"
)

const (
	DefaultBuild  = "development"
	DefaultServer = "localhost"
	DefaultRegion = "local"
	DefaultFlavor = "development"

	DefaultIdleTimeout       time.Duration = 15 * time.Second
	DefaultReadHeaderTimeout time.Duration = 0
	DefaultReadTimeout       time.Duration = 5 * time.Second
	DefaultWriteTimeout      time.Duration = 30 * time.Minute

	DefaultMaxHeaderBytes = http.DefaultMaxHeaderBytes
)

var (
	// ErrorNoPrimaryAddress is the error returned when no primary address is specified in a WebPA instance
	ErrorNoPrimaryAddress = errors.New("No primary address configured")

	// strongCipherSuites are the tls.CipherSuite values that are safe for TLS versions less than 1.3
	strongCipherSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	}
)

// executor is an internal type used to start an HTTP server.  *http.Server implements
// this interface.  It can be mocked for testing.
type executor interface {
	Serve(net.Listener) error
	ServeTLS(l net.Listener, certificateFile, keyFile string) error

	ListenAndServe() error
	ListenAndServeTLS(certificateFile, keyFile string) error

	Shutdown(ctx context.Context) error
}

func RestartableFunc(logger *zap.Logger, f func() error, errs ...error) error {
	var err error
	logger.Info("starting restartable func", zap.Errors("errors", errs))
	breakErrors := make(map[error]bool)
	for _, elem := range errs {
		breakErrors[elem] = true
	}
	for {
		err = f()
		if breakErrors[err] {
			break
		}
		logger.Debug("restartable func making a loop", zap.Error(err))
	}
	logger.Info("restartable func exiting", zap.Error(err))
	return err
}

// Serve is like ListenAndServe, but accepts a custom net.Listener
func Serve(logger *zap.Logger, l net.Listener, e executor, finalizer func()) {
	go func() {
		defer finalizer()
		logger.Error("starting server")
		// the assumption is tlsConfig has already been set
		// Note: the tlsConfig should have the certs and goodness
		logger.Error("server exited",
			zap.Error(RestartableFunc(logger, func() error { return e.Serve(l) }, http.ErrServerClosed)),
		)
	}()
}

// ListenAndServe invokes the server method
func ListenAndServe(logger *zap.Logger, e executor, finalizer func()) {
	go func() {
		defer finalizer()
		logger.Error("starting server")
		// the assumption is tlsConfig has already been set
		// Note: the tlsConfig should have the certs and goodness
		logger.Error("server exited",
			zap.Error(RestartableFunc(logger, e.ListenAndServe, http.ErrServerClosed)),
		)
	}()
}

// Basic describes a simple HTTP server.  Typically, this struct has its values
// injected via Viper.  See the New function in this package.
type Basic struct {
	Name               string
	Address            string
	CertificateFile    []string
	KeyFile            []string
	ClientCACertFile   string
	LogConnectionState bool
	MinVersion         uint16
	MaxVersion         uint16

	PeerVerifyFunc PeerVerifyCallback // Callback func to add peer client cert CN, SAN validation

	MaxConnections    int
	DisableKeepAlives bool
	MaxHeaderBytes    int
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
}

func (b *Basic) minVersion() uint16 {
	if b != nil && b.MinVersion != 0 {
		return b.MinVersion
	}

	// accept all versions
	return 0
}

func (b *Basic) maxVersion() uint16 {
	if b != nil && b.MaxVersion != 0 {
		return b.MaxVersion
	}

	// accept all versions
	return 0
}

type PeerVerifyCallback func([][]byte, [][]*x509.Certificate) error

func DefaultPeerVerifyCallback(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	// Default callback performs no validation
	return nil
}

func (b *Basic) maxConnections() int {
	if b != nil && b.MaxConnections > 0 {
		return b.MaxConnections
	}

	// no max connections set
	return 0
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

func (b *Basic) readTimeout() time.Duration {
	if b != nil && b.ReadTimeout > 0 {
		return b.ReadTimeout
	}

	return DefaultReadTimeout
}

func (b *Basic) writeTimeout() time.Duration {
	if b != nil && b.WriteTimeout > 0 {
		return b.WriteTimeout
	}

	return DefaultWriteTimeout
}

func (b *Basic) SetPeerVerifyCallback(vp PeerVerifyCallback) {
	b.PeerVerifyFunc = vp
}

// NewListener creates a decorated TCPListener appropriate for this server's configuration.
func (b *Basic) NewListener(logger *zap.Logger, activeConnections metrics.Gauge, rejectedCounter xmetrics.Adder, config *tls.Config) (net.Listener, error) {
	return xlistener.New(xlistener.Options{
		Logger:         logger,
		Address:        b.Address,
		MaxConnections: b.maxConnections(),
		Active:         activeConnections,
		Rejected:       rejectedCounter,
		Config:         config,
	})
}

func validCertSlices(certificateFiles, keyFiles []string) bool {
	valid := true
	if len(certificateFiles) > 0 && len(keyFiles) > 0 && len(certificateFiles) == len(keyFiles) {
		for i := 0; i < len(certificateFiles); i++ {
			if !(len(certificateFiles[i]) > 0 && len(certificateFiles[i]) > 0) {
				valid = false
			}
		}
	} else {
		valid = false
	}
	return valid
}

func loadCerts(certificateFiles, keyFiles []string) (certs []tls.Certificate, err error) {
	if !validCertSlices(certificateFiles, keyFiles) {
		return []tls.Certificate{}, errors.New("certFiles and keyFiles are not valid")
	}

	certs = make([]tls.Certificate, len(certificateFiles))
	for i := 0; i < len(certificateFiles); i++ {
		certs[i], err = tls.LoadX509KeyPair(certificateFiles[i], keyFiles[i])
		if err != nil {
			sallust.Default().Error("Failed to LoadX509KeyPair", zap.String("cert", certificateFiles[i]), zap.String("key", keyFiles[i]), zap.Error(err))
			return []tls.Certificate{}, err
		}
	}

	return certs, nil
}

// New creates an http.Server using this instance's configuration.  The given logger is required,
// but the handler may be nil.  If the handler is nil, http.DefaultServeMux is used, which matches
// the behavior of http.Server.
//
// This method returns nil if the configured address is empty or if any config errors occur, effectively disabling
// this server from startup.
func (b *Basic) New(logger *zap.Logger, handler http.Handler) *http.Server {
	if len(b.Address) == 0 {
		return nil
	}

	var tlsConfig *tls.Config
	if len(b.CertificateFile) > 0 && len(b.KeyFile) > 0 {
		certs, err := loadCerts(b.CertificateFile, b.KeyFile)
		if err != nil {
			logger.Error("Error loading cert and key file to configure TLS", zap.Error(err))
			return nil
		}

		tlsConfig = &tls.Config{
			Certificates: certs,
			MinVersion:   b.minVersion(),
			MaxVersion:   b.maxVersion(),

			// ensure strong ciphers when the TLS version is 1.2 or less
			CipherSuites: strongCipherSuites,
		}

		if len(b.ClientCACertFile) > 0 {
			caCert, err := ioutil.ReadFile(b.ClientCACertFile)

			if err != nil {
				logger.Error("Error loading clientCACert file to configure mTLS", zap.Error(err))
				return nil
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.ClientCAs = caCertPool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	server := &http.Server{
		Addr:              b.Address,
		Handler:           handler,
		ReadHeaderTimeout: b.readHeaderTimeout(),
		ReadTimeout:       b.readTimeout(),
		WriteTimeout:      b.writeTimeout(),
		IdleTimeout:       b.idleTimeout(),
		MaxHeaderBytes:    b.maxHeaderBytes(),
		ErrorLog:          sallust.NewServerLogger(b.Name, logger),
		TLSConfig:         tlsConfig,
		TLSNextProto:      map[string]func(*http.Server, *tls.Conn, http.Handler){}, // disable HTTP/2
	}

	if b.LogConnectionState {
		server.ConnState = sallusthttp.NewConnStateLogger(logger, zap.DebugLevel, zap.String("serverName", b.Name))
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
	CertificateFile    []string
	KeyFile            []string
	LogConnectionState bool
	HandlerOptions     promhttp.HandlerOpts
	MetricsOptions     xmetrics.Options
}

func (m *Metric) NewRegistry(modules ...xmetrics.Module) (xmetrics.Registry, error) {
	// always append the builtin server metrics, which can be overridden in configuration
	modules = append(modules, Metrics)
	return xmetrics.NewRegistry(&m.MetricsOptions, modules...)
}

func (m *Metric) New(logger *zap.Logger, chain alice.Chain, gatherer stdprometheus.Gatherer) *http.Server {
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
		ErrorLog:          sallust.NewServerLogger(m.Name, logger),
	}

	if m.LogConnectionState {
		server.ConnState = sallusthttp.NewConnStateLogger(logger, zap.DebugLevel, zap.String("serverName", m.Name))
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
	CertificateFile    []string
	KeyFile            []string
	LogConnectionState bool
	LogInterval        time.Duration
	Options            []string
}

// NewHealth creates a Health instance from this instance's configuration.  If the Address
// field is not supplied, this method returns nil.
func (h *Health) NewHealth(logger *zap.Logger, options ...health.Option) *health.Health {
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
func (h *Health) New(logger *zap.Logger, chain alice.Chain, health *health.Health) (*health.Health, *http.Server) {
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
		ErrorLog:          sallust.NewServerLogger(h.Name, logger),
	}

	if h.LogConnectionState {
		server.ConnState = sallusthttp.NewConnStateLogger(logger, zap.DebugLevel, zap.String("serverName", h.Name))
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
	Log *[]zap.Field
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
func (w *WebPA) Prepare(logger *zap.Logger, health *health.Health, registry xmetrics.Registry, primaryHandler http.Handler) (health.Monitor, concurrent.Runnable, <-chan struct{}) {
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

		activeConnections = registry.NewGauge("active_connections")
		rejectedCounter   = registry.NewCounter("rejected_connections")
		maxProcs          = registry.NewGauge("maximum_processors")

		healthHandler, healthServer = w.Health.New(logger, alice.New(staticHeaders), health)

		servers      []*http.Server
		finalizeOnce sync.Once
		done         = make(chan struct{})
		finalizer    = func() {
			finalizeOnce.Do(func() {
				defer close(done)
				for _, s := range servers {
					logger.Error("finalizing server", zap.Error(s.Close()))
				}
			})
		}
	)

	return healthHandler, concurrent.RunnableFunc(func(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
		primaryHandler = staticHeaders(w.decorateWithBasicMetrics(registry, primaryHandler))

		// create all the servers first, so that we can populate the servers slice
		// without worrying about concurrency
		primaryServer := w.Primary.New(logger, primaryHandler)
		if primaryServer == nil {
			// the primary server is required
			close(done)
			return ErrorNoPrimaryAddress
		}

		alternateServer := w.Alternate.New(logger, primaryHandler)
		if alternateServer != nil {
			servers = append(servers, alternateServer)
		}

		if healthServer != nil {
			servers = append(servers, healthServer)
		}

		pprofServer := w.Pprof.New(logger, nil)
		if pprofServer != nil {
			servers = append(servers, pprofServer)
		}

		metricsServer := w.Metric.New(logger, alice.New(staticHeaders), registry)
		if metricsServer != nil {
			servers = append(servers, metricsServer)
		}

		// create any necessary listeners first, so that we return early if errors occur

		primaryLogger := logger.With(zap.String("serverName", w.Primary.Name), zap.String("bindAddress", w.Primary.Address))
		primaryListener, err := w.Primary.NewListener(
			primaryLogger,
			activeConnections.With("server", "primary"),
			rejectedCounter.With("server", "primary"),
			primaryServer.TLSConfig,
		)

		if err != nil {
			close(done)
			return err
		}

		// now we can start all the servers

		// start the alternate server first, so we can short-circuit in the case of errors
		if alternateServer != nil {
			alternateLogger := logger.With(zap.String("serverName", w.Alternate.Name), zap.String("bindAddress", w.Alternate.Address))
			alternateListener, err := w.Alternate.NewListener(
				alternateLogger,
				activeConnections.With("server", "alternate"),
				rejectedCounter.With("server", "alternate"),
				alternateServer.TLSConfig,
			)

			if err != nil {
				close(done)
				return err
			}

			Serve(alternateLogger, alternateListener, alternateServer, finalizer)
		}

		Serve(primaryLogger, primaryListener, primaryServer, finalizer)

		if healthHandler != nil && healthServer != nil {
			ListenAndServe(logger.With(zap.String("serverName", w.Health.Name), zap.String("bindAddress", w.Health.Address)), healthServer, finalizer)
			healthHandler.Run(waitGroup, shutdown)
		}

		if pprofServer != nil {
			ListenAndServe(
				logger.With(zap.String("serverName", w.Pprof.Name), zap.String("bindAddress", w.Pprof.Address)),
				pprofServer,
				finalizer,
			)
		}

		if metricsServer != nil {
			ListenAndServe(
				logger.With(zap.String("serverName", w.Metric.Name), zap.String("bindAddress", w.Metric.Address)),
				metricsServer,
				finalizer,
			)
		}

		// Output, to metrics, the maximum number of CPUs available to this process
		maxProcs.Set(float64(runtime.GOMAXPROCS(0)))

		return nil
	}), done
}

// decorateWithBasicMetrics wraps a WebPA server handler with basic instrumentation metrics
func (w *WebPA) decorateWithBasicMetrics(p xmetrics.PrometheusProvider, next http.Handler) http.Handler {
	var (
		requestCounter    = p.NewCounterVec(APIRequestsTotal)
		inFlight          = p.NewGaugeVec(InFlightRequests).WithLabelValues()
		requestDuration   = p.NewHistogramVec(RequestDurationSeconds)
		requestSize       = p.NewHistogramVec(RequestSizeBytes)
		responseSizeVec   = p.NewHistogramVec(ResponseSizeBytes)
		timeToWriteHeader = p.NewHistogramVec(TimeWritingHeaderSeconds)
	)

	//todo: Example documentation does something interesting with /pull vs. /push endpoints
	//https://godoc.org/github.com/prometheus/client_golang/prometheus/promhttp#InstrumentHandlerDuration
	//for now, let's keep it simple so /metrics only

	return promhttp.InstrumentHandlerInFlight(inFlight,
		promhttp.InstrumentHandlerCounter(requestCounter,
			promhttp.InstrumentHandlerDuration(requestDuration,
				promhttp.InstrumentHandlerResponseSize(responseSizeVec,
					promhttp.InstrumentHandlerRequestSize(requestSize,
						promhttp.InstrumentHandlerTimeToWriteHeader(timeToWriteHeader, next))),
			),
		),
	)
}
