package server

import (
	"bytes"
	"github.com/Comcast/webpa-common/logging"
	"net/http"
	"testing"
)

type builderExpect struct {
	name            string
	address         string
	certificateFile string
	keyFile         string
	logger          logging.Logger
	handler         http.Handler
}

func (expect *builderExpect) assert(t *testing.T, logBuffer *bytes.Buffer, builder *Builder) {
	actual := builder.Build()
	if actual == nil {
		t.Fatal("Build() returned nil")
	}

	if expect.name != actual.name {
		t.Errorf("Expected name %s, but got %s", expect.name, actual.name)
	}

	if expect.address != actual.address {
		t.Errorf("Expected address %s, but got %s", expect.name, actual.name)
	}

	if expect.certificateFile != actual.certificateFile {
		t.Errorf("Expected certificate file %s, but got %s", expect.certificateFile, actual.certificateFile)
	}

	if expect.keyFile != actual.keyFile {
		t.Errorf("Expected key file %s, but got %s", expect.keyFile, actual.keyFile)
	}

	if expect.logger != actual.logger {
		t.Errorf("Expected logger %#v, but got %#v", expect.logger, actual.logger)
	}

	logBuffer.Reset()
	actual.logger.Debug("verify logger")
	if logBuffer.Len() == 0 {
		t.Error("Build() did not use the configured logger")
	}

	if httpServer, ok := actual.serverExecutor.(*http.Server); !ok {
		t.Errorf("Build() did not produce an http.Server")
	} else {
		if expect.address != httpServer.Addr {
			t.Errorf("Expected address %s, but got %s", expect.name, httpServer.Addr)
		}

		if httpServer.ErrorLog == nil {
			t.Error("Build() did not produce an ErrorLog")
		} else {
			logBuffer.Reset()
			httpServer.ErrorLog.Println("verify ErrorLog")
			if logBuffer.Len() == 0 {
				t.Error("The ErrorLog did not use the configured logger")
			}
		}

		if httpServer.ConnState == nil {
			t.Error("Build() did not produce a ConnState function")
		} else {
			logBuffer.Reset()
			httpServer.ConnState(mockConn{}, http.StateActive)
			if logBuffer.Len() == 0 {
				t.Error("The ConnState function does not use the configured logger")
			}
		}
	}
}

func TestBuilder(t *testing.T) {
	var logBuffer bytes.Buffer
	expectedLogger := &logging.LoggerWriter{&logBuffer}

	var testData = []struct {
		builder Builder
		expect  builderExpect
	}{
		{
			Builder{
				Logger: expectedLogger,
			},
			builderExpect{
				logger: expectedLogger,
			},
		},
	}

	for _, record := range testData {
		record.expect.assert(t, &logBuffer, &record.builder)
	}
}
