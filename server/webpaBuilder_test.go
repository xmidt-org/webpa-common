package server

import (
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/types"
	"net/http"
	"os"
	"testing"
	"time"
)

type webpaBuilderExpect struct {
	serverName          string
	primaryAddress      string
	healthAddress       string
	healthCheckInterval time.Duration
	pprofAddress        string
	certificateFile     string
	keyFile             string
}

var webpaBuilderTestData = []struct {
	builder WebPABuilder
	expect  webpaBuilderExpect
}{
	{
		builder: WebPABuilder{},
		expect: webpaBuilderExpect{
			serverName:          DefaultServerName,
			primaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			healthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			healthCheckInterval: DefaultHealthCheckInterval,
			pprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
	},
	{
		builder: WebPABuilder{
			Configuration: &Configuration{},
		},
		expect: webpaBuilderExpect{
			serverName:          DefaultServerName,
			primaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			healthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			healthCheckInterval: DefaultHealthCheckInterval,
			pprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
	},
	{
		builder: WebPABuilder{
			Configuration: &Configuration{
				ServerName: "onlyoneset",
			},
		},
		expect: webpaBuilderExpect{
			serverName:          "onlyoneset",
			primaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			healthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			healthCheckInterval: DefaultHealthCheckInterval,
			pprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
	},
	{
		builder: WebPABuilder{
			Configuration: &Configuration{
				Port: 2857,
			},
		},
		expect: webpaBuilderExpect{
			serverName:          DefaultServerName,
			primaryAddress:      ":2857",
			healthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			healthCheckInterval: DefaultHealthCheckInterval,
			pprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
	},
	{
		builder: WebPABuilder{
			Configuration: &Configuration{
				HealthCheckPort: 83,
			},
		},
		expect: webpaBuilderExpect{
			serverName:          DefaultServerName,
			primaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			healthAddress:       ":83",
			healthCheckInterval: DefaultHealthCheckInterval,
			pprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
	},
	{
		builder: WebPABuilder{
			Configuration: &Configuration{
				HealthCheckInterval: types.Duration(time.Hour * 5),
			},
		},
		expect: webpaBuilderExpect{
			serverName:          DefaultServerName,
			primaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			healthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			healthCheckInterval: time.Hour * 5,
			pprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
	},
	{
		builder: WebPABuilder{
			Configuration: &Configuration{
				PprofPort: 2395,
			},
		},
		expect: webpaBuilderExpect{
			serverName:          DefaultServerName,
			primaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			healthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			healthCheckInterval: DefaultHealthCheckInterval,
			pprofAddress:        ":2395",
		},
	},
	{
		builder: WebPABuilder{
			Configuration: &Configuration{
				ServerName:          "foobar",
				Port:                1281,
				HealthCheckPort:     56001,
				HealthCheckInterval: types.Duration(time.Minute * 3412),
				PprofPort:           41508,
			},
		},
		expect: webpaBuilderExpect{
			serverName:          "foobar",
			primaryAddress:      ":1281",
			healthAddress:       ":56001",
			healthCheckInterval: time.Minute * 3412,
			pprofAddress:        ":41508",
		},
	},
	{
		builder: WebPABuilder{
			Configuration: &Configuration{
				ServerName:          "groograar",
				Port:                8347,
				HealthCheckPort:     81,
				HealthCheckInterval: types.Duration(time.Minute * 797),
				PprofPort:           55692,
				CertificateFile:     "/etc/groograar/cert",
				KeyFile:             "/etc/groograar/key",
			},
		},
		expect: webpaBuilderExpect{
			serverName:          "groograar",
			primaryAddress:      ":8347",
			healthAddress:       ":81",
			healthCheckInterval: time.Minute * 797,
			pprofAddress:        ":55692",
			certificateFile:     "/etc/groograar/cert",
			keyFile:             "/etc/groograar/key",
		},
	},
}

func TestWebPABuilderConfiguration(t *testing.T) {
	for _, record := range webpaBuilderTestData {
		actualServerName := record.builder.ServerName()
		if record.expect.serverName != actualServerName {
			t.Errorf("Expected server name %s, but got %s", record.expect.serverName, actualServerName)
		}

		actualPrimaryAddress := record.builder.PrimaryAddress()
		if record.expect.primaryAddress != actualPrimaryAddress {
			t.Errorf("Expected primary address %s, but got %s", record.expect.primaryAddress, actualPrimaryAddress)
		}

		actualHealthAddress := record.builder.HealthAddress()
		if record.expect.healthAddress != actualHealthAddress {
			t.Errorf("Expected health address %s, but got %s", record.expect.healthAddress, actualHealthAddress)
		}

		actualHealthCheckInterval := record.builder.HealthCheckInterval()
		if record.expect.healthCheckInterval != actualHealthCheckInterval {
			t.Errorf("Expected health check interval %s, but got %s", record.expect.healthCheckInterval, actualHealthCheckInterval)
		}

		actualPprofAddress := record.builder.PprofAddress()
		if record.expect.pprofAddress != actualPprofAddress {
			t.Errorf("Expected primary address %s, but got %s", record.expect.pprofAddress, actualPprofAddress)
		}
	}
}

func TestBuildPrimary(t *testing.T) {
	for _, record := range webpaBuilderTestData {
		expectedLogger := &logging.LoggerWriter{os.Stdout}
		handlerCalled := false
		expectedHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			handlerCalled = true
		})

		builder := record.builder
		builder.PrimaryHandler = expectedHandler
		builder.LoggerFactory = &testLoggerFactory{
			t,
			func(t *testing.T, name string) (logging.Logger, error) {
				if record.expect.serverName != name {
					t.Fatalf("Expected logger name %s, but got %s", record.expect.serverName, name)
				}

				return expectedLogger, nil
			},
		}

		runnable, err := builder.BuildPrimary()
		if err != nil {
			t.Fatalf("BuildPrimary() failed: %v", err)
		}

		primary, ok := runnable.(*webPA)
		if !ok {
			t.Fatal("BuildPrimary() did not return a webPA")
		}

		if record.expect.serverName != primary.name {
			t.Errorf("Expected server name %s, but got %s", record.expect.serverName, primary.name)
		}

		if expectedLogger != primary.logger {
			t.Errorf("Expected logger %#v, but got %#v", expectedLogger, primary.logger)
		}

		if record.expect.primaryAddress != primary.address {
			t.Errorf("Expected primary address %s, but got %s", record.expect.primaryAddress, primary.address)
		}

		if record.expect.certificateFile != primary.certificateFile {
			t.Errorf("Expected certificate file %s, but got %s", record.expect.certificateFile, primary.certificateFile)
		}

		if record.expect.keyFile != primary.keyFile {
			t.Errorf("Expected key file %s, but got %s", record.expect.keyFile, primary.keyFile)
		}

		httpServer, ok := primary.serverExecutor.(*http.Server)
		if !ok {
			t.Fatal("BuildPrimary() did not generate an http.Server")
		}

		if record.expect.primaryAddress != httpServer.Addr {
			t.Errorf("Expected http.Server address %s, but got %s", record.expect.primaryAddress, httpServer.Addr)
		}

		httpServer.Handler.ServeHTTP(nil, nil)
		if !handlerCalled {
			t.Error("BuildPrimary() did not use the supplied handler")
		}

		if httpServer.ConnState == nil {
			t.Error("BuildPrimary() did not generate a ConnState function")
		}

		if httpServer.ErrorLog == nil {
			t.Error("BuildPrimary() did not generate an ErrorLog")
		}
	}
}

func TestBuildPprofUsingDefaultServeMux(t *testing.T) {
	for _, record := range webpaBuilderTestData {
		expectedServerName := record.expect.serverName + pprofSuffix
		expectedLogger := &logging.LoggerWriter{os.Stdout}
		builder := record.builder
		builder.LoggerFactory = &testLoggerFactory{
			t,
			func(t *testing.T, name string) (logging.Logger, error) {
				if expectedServerName != name {
					t.Fatalf("Expected logger name %s, but got %s", expectedServerName, name)
				}

				return expectedLogger, nil
			},
		}

		runnable, err := builder.BuildPprof()
		if err != nil {
			t.Fatalf("BuildPprof() failed: %v", err)
		}

		pprof, ok := runnable.(*webPA)
		if !ok {
			t.Fatal("BuildPprof() did not return a webPA")
		}

		if expectedServerName != pprof.name {
			t.Errorf("Expected server name %s, but got %s", expectedServerName, pprof.name)
		}

		if expectedLogger != pprof.logger {
			t.Errorf("Expected logger %#v, but got %#v", expectedLogger, pprof.logger)
		}

		if record.expect.pprofAddress != pprof.address {
			t.Errorf("Expected pprof address %s, but got %s", record.expect.pprofAddress, pprof.address)
		}

		if len(pprof.certificateFile) != 0 {
			t.Errorf("BuildPprof() used certificate file %s", pprof.certificateFile)
		}

		if len(pprof.keyFile) != 0 {
			t.Errorf("BuildPprof() used key file %s", pprof.certificateFile)
		}

		httpServer, ok := pprof.serverExecutor.(*http.Server)
		if !ok {
			t.Fatal("BuildPprof() did not generate an http.Server")
		}

		if record.expect.pprofAddress != httpServer.Addr {
			t.Errorf("Expected http.Server address %s, but got %s", record.expect.pprofAddress, httpServer.Addr)
		}

		if http.DefaultServeMux != httpServer.Handler {
			t.Error("BuildPprof() did not use http.DefaultServeMux")
		}

		if httpServer.ConnState == nil {
			t.Error("BuildPprof() did not generate a ConnState function")
		}

		if httpServer.ErrorLog == nil {
			t.Error("BuildPprof() did not generate an ErrorLog")
		}
	}
}

func TestBuildPprofUsingCustomHandler(t *testing.T) {
	for _, record := range webpaBuilderTestData {
		expectedServerName := record.expect.serverName + pprofSuffix
		handlerCalled := false
		expectedHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			handlerCalled = true
		})

		expectedLogger := &logging.LoggerWriter{os.Stdout}
		builder := record.builder
		builder.PprofHandler = expectedHandler
		builder.LoggerFactory = &testLoggerFactory{
			t,
			func(t *testing.T, name string) (logging.Logger, error) {
				if expectedServerName != name {
					t.Fatalf("Expected logger name %s, but got %s", expectedServerName, name)
				}

				return expectedLogger, nil
			},
		}

		runnable, err := builder.BuildPprof()
		if err != nil {
			t.Fatalf("BuildPprof() failed: %v", err)
		}

		pprof, ok := runnable.(*webPA)
		if !ok {
			t.Fatal("BuildPprof() did not return a webPA")
		}

		if expectedServerName != pprof.name {
			t.Errorf("Expected server name %s, but got %s", expectedServerName, pprof.name)
		}

		if expectedLogger != pprof.logger {
			t.Errorf("Expected logger %#v, but got %#v", expectedLogger, pprof.logger)
		}

		if record.expect.pprofAddress != pprof.address {
			t.Errorf("Expected pprof address %s, but got %s", record.expect.pprofAddress, pprof.address)
		}

		if len(pprof.certificateFile) != 0 {
			t.Errorf("BuildPprof() used certificate file %s", pprof.certificateFile)
		}

		if len(pprof.keyFile) != 0 {
			t.Errorf("BuildPprof() used key file %s", pprof.certificateFile)
		}

		httpServer, ok := pprof.serverExecutor.(*http.Server)
		if !ok {
			t.Fatal("BuildPprof() did not generate an http.Server")
		}

		if record.expect.pprofAddress != httpServer.Addr {
			t.Errorf("Expected http.Server address %s, but got %s", record.expect.pprofAddress, httpServer.Addr)
		}

		httpServer.Handler.ServeHTTP(nil, nil)
		if !handlerCalled {
			t.Error("BuildPprof() did not use the supplied handler")
		}

		if httpServer.ConnState == nil {
			t.Error("BuildPprof() did not generate a ConnState function")
		}

		if httpServer.ErrorLog == nil {
			t.Error("BuildPprof() did not generate an ErrorLog")
		}
	}
}

func TestBuildHealth(t *testing.T) {
	for _, record := range webpaBuilderTestData {
		expectedServerName := record.expect.serverName + healthSuffix
		expectedLogger := &logging.LoggerWriter{os.Stdout}
		handlerCalled := false
		expectedHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			handlerCalled = true
		})

		builder := record.builder
		builder.HealthHandler = expectedHandler
		builder.LoggerFactory = &testLoggerFactory{
			t,
			func(t *testing.T, name string) (logging.Logger, error) {
				if expectedServerName != name {
					t.Fatalf("Expected logger name %s, but got %s", expectedServerName, name)
				}

				return expectedLogger, nil
			},
		}

		runnable, err := builder.BuildHealth()
		if err != nil {
			t.Fatalf("BuildHealth() failed: %v", err)
		}

		health, ok := runnable.(*webPA)
		if !ok {
			t.Fatal("BuildHealth() did not return a webPA")
		}

		if expectedServerName != health.name {
			t.Errorf("Expected server name %s, but got %s", record.expect.serverName, health.name)
		}

		if expectedLogger != health.logger {
			t.Errorf("Expected logger %#v, but got %#v", expectedLogger, health.logger)
		}

		if record.expect.healthAddress != health.address {
			t.Errorf("Expected health address %s, but got %s", record.expect.primaryAddress, health.address)
		}

		if len(health.certificateFile) != 0 {
			t.Errorf("BuildHealth() used certificate file %s", health.certificateFile)
		}

		if len(health.keyFile) != 0 {
			t.Errorf("BuildHealth() used key file %s", health.certificateFile)
		}

		httpServer, ok := health.serverExecutor.(*http.Server)
		if !ok {
			t.Fatal("BuildHealth() did not generate an http.Server")
		}

		if record.expect.healthAddress != httpServer.Addr {
			t.Errorf("Expected http.Server address %s, but got %s", record.expect.healthAddress, httpServer.Addr)
		}

		httpServer.Handler.ServeHTTP(nil, nil)
		if !handlerCalled {
			t.Error("BuildHealth() did not use the supplied handler")
		}

		if httpServer.ConnState == nil {
			t.Error("BuildHealth() did not generate a ConnState function")
		}

		if httpServer.ErrorLog == nil {
			t.Error("BuildHealth() did not generate an ErrorLog")
		}
	}
}
