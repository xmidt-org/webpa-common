package server

import (
	"bytes"
	"fmt"
	"github.com/Comcast/webpa-common/concurrent"
	"net/http"
	"testing"
)

// dummyHandler is an http.Handler used to verify that builder code worked
type dummyHandler struct {
}

func (d dummyHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
}

type webpaExpect struct {
	name            string
	address         string
	handler         http.Handler
	certificateFile string
	keyFile         string
}

func (expect *webpaExpect) assertBuildFunc(t *testing.T, loggingBuffer *bytes.Buffer, buildFunc func() (concurrent.Runnable, error)) {
	product, err := buildFunc()
	if err != nil {
		t.Fatalf("The builder function failed: %v", err)
	}

	expect.assertProduct(t, loggingBuffer, product)
}

func (expect *webpaExpect) assertProduct(t *testing.T, loggingBuffer *bytes.Buffer, product concurrent.Runnable) {
	actual, ok := product.(*webPA)
	if !ok {
		t.Fatal("The builder product is not a webPA instance")
	}

	if expect.name != actual.name {
		t.Errorf("Expected name %s, but got %s", expect.name, actual.name)
	}

	if expect.address != actual.address {
		t.Errorf("Expected address %s, but got %s", expect.address, actual.address)
	}

	if expect.certificateFile != actual.certificateFile {
		t.Errorf("Expected certificateFile %s, but got %s", expect.certificateFile, actual.certificateFile)
	}

	if expect.keyFile != actual.keyFile {
		t.Errorf("Expected keyFile %s, but got %s", expect.keyFile, actual.keyFile)
	}

	if actual.logger == nil {
		t.Error("No logger set on the webPA instance")
	} else {
		loggingBuffer.Reset()
		actual.logger.Debug("test test test")
		if loggingBuffer.Len() == 0 {
			t.Error("Incorrect logger set on webPA instance")
		}
	}

	httpServer, ok := actual.serverExecutor.(*http.Server)
	if !ok {
		t.Fatal("The serverExecutor is not an http.Server")
	}

	if expect.address != httpServer.Addr {
		t.Errorf("Expected httpServer address %s, but got %s", expect.address, httpServer.Addr)
	}

	if expect.handler != httpServer.Handler {
		t.Errorf("Expected httpServer handler %v, but got %v", expect.handler, httpServer.Handler)
	}

	if httpServer.ErrorLog == nil {
		t.Error("No ErrorLog set on httpServer")
	} else {
		loggingBuffer.Reset()
		httpServer.ErrorLog.Println("test test test")
		if loggingBuffer.Len() == 0 {
			t.Error("Incorrect ErrorLog set on webPA instance")
		}
	}

	if httpServer.ConnState == nil {
		t.Error("No ConnState set on httpServer")
	} else {
		loggingBuffer.Reset()
		httpServer.ConnState(mockConn{t}, http.StateNew)
		if loggingBuffer.Len() == 0 {
			t.Error("Incorrect ConnState set on webPA instance")
		}
	}
}

var (
	primaryHandler     = dummyHandler{}
	healthHandler      = dummyHandler{}
	customPprofHandler = dummyHandler{}

	webpaBuilderTestData = []struct {
		builder       WebPABuilder
		expectPrimary webpaExpect
		expectPprof   webpaExpect
		expectHealth  webpaExpect
	}{
		{
			builder: WebPABuilder{
				PrimaryHandler: primaryHandler,
				HealthHandler:  healthHandler,
			},
			expectPrimary: webpaExpect{
				name:    DefaultServerName,
				address: fmt.Sprintf(":%d", DefaultPort),
				handler: primaryHandler,
			},
			expectHealth: webpaExpect{
				name:    DefaultServerName + healthSuffix,
				address: fmt.Sprintf(":%d", DefaultHealthCheckPort),
				handler: healthHandler,
			},
			expectPprof: webpaExpect{
				name:    DefaultServerName + pprofSuffix,
				address: fmt.Sprintf(":%d", DefaultPprofPort),
				handler: http.DefaultServeMux,
			},
		},
		{
			builder: WebPABuilder{
				Configuration:  &Configuration{},
				PrimaryHandler: primaryHandler,
				HealthHandler:  healthHandler,
			},
			expectPrimary: webpaExpect{
				name:    DefaultServerName,
				address: fmt.Sprintf(":%d", DefaultPort),
				handler: primaryHandler,
			},
			expectHealth: webpaExpect{
				name:    DefaultServerName + healthSuffix,
				address: fmt.Sprintf(":%d", DefaultHealthCheckPort),
				handler: healthHandler,
			},
			expectPprof: webpaExpect{
				name:    DefaultServerName + pprofSuffix,
				address: fmt.Sprintf(":%d", DefaultPprofPort),
				handler: http.DefaultServeMux,
			},
		},
		{
			builder: WebPABuilder{
				Configuration: &Configuration{
					ServerName:      "foobar",
					Port:            2739,
					HealthCheckPort: 1842,
					PprofPort:       28391,
				},
				PrimaryHandler: primaryHandler,
				HealthHandler:  healthHandler,
			},
			expectPrimary: webpaExpect{
				name:    "foobar",
				address: ":2739",
				handler: primaryHandler,
			},
			expectHealth: webpaExpect{
				name:    "foobar" + healthSuffix,
				address: ":1842",
				handler: healthHandler,
			},
			expectPprof: webpaExpect{
				name:    "foobar" + pprofSuffix,
				address: ":28391",
				handler: http.DefaultServeMux,
			},
		},
		{
			builder: WebPABuilder{
				Configuration: &Configuration{
					ServerName:      "graarpants",
					Port:            29019,
					HealthCheckPort: 28,
					PprofPort:       129,
				},
				PrimaryHandler: primaryHandler,
				HealthHandler:  healthHandler,
				PprofHandler:   customPprofHandler,
			},
			expectPrimary: webpaExpect{
				name:    "graarpants",
				address: ":29019",
				handler: primaryHandler,
			},
			expectHealth: webpaExpect{
				name:    "graarpants" + healthSuffix,
				address: ":28",
				handler: healthHandler,
			},
			expectPprof: webpaExpect{
				name:    "graarpants" + pprofSuffix,
				address: ":129",
				handler: customPprofHandler,
			},
		},
	}
)

func TestWebPABuilderConfiguration(t *testing.T) {
	for _, record := range webpaBuilderTestData {
		builder := record.builder
		actualServerName := builder.ServerName()
		if record.expectPrimary.name != record.builder.ServerName() {
			t.Errorf("Expected server name %s, but got %s", record.expectPrimary.name, actualServerName)
		}

		actualPrimaryAddress := builder.PrimaryAddress()
		if record.expectPrimary.address != actualPrimaryAddress {
			t.Errorf("Expected primary address %s, but got %s", record.expectPrimary.address, actualPrimaryAddress)
		}

		actualHealthAddress := builder.HealthAddress()
		if record.expectHealth.address != actualHealthAddress {
			t.Errorf("Expected health address %s, but got %s", record.expectHealth.address, actualHealthAddress)
		}

		actualPprofAddress := builder.PprofAddress()
		if record.expectPprof.address != actualPprofAddress {
			t.Errorf("Expected pprof address %s, but got %s", record.expectPprof.address, actualPprofAddress)
		}
	}
}

func TestBuildPrimary(t *testing.T) {
	for _, record := range webpaBuilderTestData {
		expect := record.expectPrimary
		builder := record.builder
		loggerFactory := newTestLoggerFactory(t, expect.name)
		builder.LoggerFactory = loggerFactory

		expect.assertBuildFunc(t, &loggerFactory.buffer, builder.BuildPrimary)
	}
}

func TestBuildPprof(t *testing.T) {
	for _, record := range webpaBuilderTestData {
		expect := record.expectPprof
		builder := record.builder
		loggerFactory := newTestLoggerFactory(t, expect.name)
		builder.LoggerFactory = loggerFactory

		expect.assertBuildFunc(t, &loggerFactory.buffer, builder.BuildPprof)
	}
}

func TestBuildHealth(t *testing.T) {
	for _, record := range webpaBuilderTestData {
		expect := record.expectHealth
		builder := record.builder
		loggerFactory := newTestLoggerFactory(t, expect.name)
		builder.LoggerFactory = loggerFactory

		expect.assertBuildFunc(t, &loggerFactory.buffer, builder.BuildHealth)
	}
}

func TestBuildAll(t *testing.T) {
	for _, record := range webpaBuilderTestData {
		builder := record.builder
		loggerFactory := newTestLoggerFactory(t, record.expectPrimary.name, record.expectPprof.name, record.expectHealth.name)
		builder.LoggerFactory = loggerFactory
		if runnableSet, err := builder.BuildAll(); err != nil {
			t.Fatalf("BuildAll() failed: %v", err)
		} else {
			if len(runnableSet) != 3 {
				t.Fatalf("Expected count of runnables to be 3, but got %d", len(runnableSet))
			}

			// the order is important: we want primary started last
			record.expectPprof.assertProduct(t, &loggerFactory.buffer, runnableSet[0])
			record.expectHealth.assertProduct(t, &loggerFactory.buffer, runnableSet[1])
			record.expectPrimary.assertProduct(t, &loggerFactory.buffer, runnableSet[2])
		}

	}
}
