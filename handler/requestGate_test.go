package handler

import (
	"bytes"
	"github.com/Comcast/webpa-common/fact"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"testing"
)

type testConnection struct {
	wasCalled bool
	result    bool
}

func (t *testConnection) Connected() bool {
	t.wasCalled = true
	return t.result
}

func TestMergeConnections(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		testConnections []testConnection
		expected        bool
	}{
		{
			[]testConnection{},
			true,
		},
		{
			[]testConnection{
				testConnection{result: true},
			},
			true,
		},
		{
			[]testConnection{
				testConnection{result: true},
				testConnection{result: true},
			},
			true,
		},
		{
			[]testConnection{
				testConnection{result: false},
			},
			false,
		},
		{
			[]testConnection{
				testConnection{result: false},
				testConnection{result: true},
			},
			false,
		},
		{
			[]testConnection{
				testConnection{result: true},
				testConnection{result: false},
			},
			false,
		},
		{
			[]testConnection{
				testConnection{result: false},
				testConnection{result: true},
				testConnection{result: true},
			},
			false,
		},
		{
			[]testConnection{
				testConnection{result: true},
				testConnection{result: false},
				testConnection{result: true},
			},
			false,
		},
		{
			[]testConnection{
				testConnection{result: true},
				testConnection{result: true},
				testConnection{result: false},
			},
			false,
		},
	}

	for _, record := range testData {
		connections := make([]Connection, len(record.testConnections))
		for index := 0; index < len(record.testConnections); index++ {
			connections[index] = &record.testConnections[index]
		}

		merged := MergeConnections(connections...)
		assert.Equal(record.expected, merged.Connected())

		expectCalled := true
		for _, testConnection := range record.testConnections {
			assert.Equal(expectCalled, testConnection.wasCalled)
			if expectCalled && !testConnection.result {
				expectCalled = false
			}
		}
	}
}

func TestRequestGate(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		connection                 Connection
		unavailableStatus          int
		unavailableMessage         string
		expectContextHandlerCalled bool
	}{
		{
			&testConnection{result: true},
			555,
			"foobar",
			true,
		},
		{
			&testConnection{result: false},
			555,
			"foobar",
			false,
		},
	}

	for _, record := range testData {
		{
			t.Log("Empty context")
			contextHandler := &testContextHandler{assert: assert}
			requestGate := RequestGate(record.connection, record.unavailableStatus, record.unavailableMessage)
			response, request := dummyHttpOperation()
			requestGate.ServeHTTP(context.Background(), response, request, contextHandler)

			assert.Equal(record.expectContextHandlerCalled, contextHandler.wasCalled)

			if !record.expectContextHandlerCalled {
				assertJsonErrorResponse(assert, response, record.unavailableStatus, record.unavailableMessage)
			}
		}

		{
			t.Log("With logger in context")
			var output bytes.Buffer
			logger := &logging.LoggerWriter{&output}
			ctx := fact.SetLogger(context.Background(), logger)

			contextHandler := &testContextHandler{assert: assert}
			requestGate := RequestGate(record.connection, record.unavailableStatus, record.unavailableMessage)
			response, request := dummyHttpOperation()
			requestGate.ServeHTTP(ctx, response, request, contextHandler)

			assert.Equal(record.expectContextHandlerCalled, contextHandler.wasCalled)

			if !record.expectContextHandlerCalled {
				assertJsonErrorResponse(assert, response, record.unavailableStatus, record.unavailableMessage)
				assert.NotEmpty(output.Bytes())
			}
		}
	}
}
