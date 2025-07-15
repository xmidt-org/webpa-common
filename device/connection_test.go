// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"errors"
	"testing"
	"time"

	"github.com/go-kit/kit/metrics/generic"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

func TestNewDeadline(t *testing.T) {
	t.Run("NoTimeout", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)

			now = func() time.Time {
				assert.Fail("now should not be called")
				return time.Time{}
			}

			deadline = NewDeadline(-1, now)
		)

		require.NotNil(deadline)
		assert.True(deadline().IsZero())
	})

	t.Run("WithTimeout", func(t *testing.T) {
		t.Run("CustomNow", func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				expectedTime    = time.Now()
				expectedTimeout = 15 * time.Minute
				deadline        = NewDeadline(expectedTimeout, func() time.Time { return expectedTime })
			)

			require.NotNil(deadline)
			actualTime := deadline()
			assert.Equal(expectedTime.Add(expectedTimeout), actualTime)
		})

		t.Run("DefaultNow", func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				baseTime        = time.Now()
				expectedTimeout = 17 * time.Second
				deadline        = NewDeadline(expectedTimeout, nil)
			)

			require.NotNil(deadline)
			actualTime := deadline()
			assert.True(actualTime.Sub(baseTime) >= expectedTimeout)
		})
	})
}

func TestSetPongHandler(t *testing.T) {
	t.Run("NoError", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)

			now     = time.Now()
			reader  = new(mockConnectionReader)
			counter = generic.NewCounter("test")

			pongHandler func(string) error
		)

		// nolint: typecheck
		reader.On("SetPongHandler", mock.MatchedBy(func(func(string) error) bool { return true })).
			Run(func(arguments mock.Arguments) {
				pongHandler = arguments.Get(0).(func(string) error)
			}).
			Once()
		// nolint: typecheck
		reader.On("SetReadDeadline", now).Return((error)(nil)).Once()

		SetPongHandler(reader, xmetrics.NewIncrementer(counter), func() time.Time { return now })
		require.NotNil(pongHandler)
		assert.NoError(pongHandler("does not matter"))
		assert.Equal(1.0, counter.Value())

		// nolint: typecheck
		reader.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)

			now           = time.Now()
			expectedError = errors.New("expected")
			reader        = new(mockConnectionReader)
			counter       = generic.NewCounter("test")

			pongHandler func(string) error
		)

		// nolint: typecheck
		reader.On("SetPongHandler", mock.MatchedBy(func(func(string) error) bool { return true })).
			Run(func(arguments mock.Arguments) {
				pongHandler = arguments.Get(0).(func(string) error)
			}).
			Once()
		// nolint: typecheck
		reader.On("SetReadDeadline", now).Return(expectedError).Once()

		SetPongHandler(reader, xmetrics.NewIncrementer(counter), func() time.Time { return now })
		require.NotNil(pongHandler)
		assert.Equal(expectedError, pongHandler("does not matter"))
		assert.Equal(1.0, counter.Value())

		// nolint: typecheck
		reader.AssertExpectations(t)
	})
}

func TestNewPinger(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)

			writer           = new(mockConnectionWriter)
			counter          = generic.NewCounter("test")
			expectedDeadline = time.Now()
		)

		pinger, err := NewPinger(writer, xmetrics.NewIncrementer(counter), []byte("ping data"), func() time.Time { return expectedDeadline })
		assert.NoError(err)
		require.NotNil(pinger)
		// nolint: typecheck
		writer.On("SetWriteDeadline", expectedDeadline).Return((error)(nil)).Once()
		// nolint: typecheck
		writer.On("WritePreparedMessage", mock.MatchedBy(func(*websocket.PreparedMessage) bool { return true })).Return((error)(nil)).Once()

		assert.NoError(pinger())
		assert.Equal(1.0, counter.Value())

		// nolint: typecheck
		writer.AssertExpectations(t)
	})

	t.Run("SetWriteDeadlineError", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)

			writer           = new(mockConnectionWriter)
			counter          = generic.NewCounter("test")
			expectedDeadline = time.Now()
			expectedError    = errors.New("expected")
		)

		pinger, err := NewPinger(writer, xmetrics.NewIncrementer(counter), []byte("ping data"), func() time.Time { return expectedDeadline })
		assert.NoError(err)
		require.NotNil(pinger)
		// nolint: typecheck
		writer.On("SetWriteDeadline", expectedDeadline).Return(expectedError).Once()

		assert.Equal(expectedError, pinger())
		assert.Zero(counter.Value())

		// nolint: typecheck
		writer.AssertExpectations(t)
	})

	t.Run("WritePreparedMessageError", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)

			writer           = new(mockConnectionWriter)
			counter          = generic.NewCounter("test")
			expectedDeadline = time.Now()
			expectedError    = errors.New("expected")
		)

		pinger, err := NewPinger(writer, xmetrics.NewIncrementer(counter), []byte("ping data"), func() time.Time { return expectedDeadline })
		assert.NoError(err)
		require.NotNil(pinger)
		// nolint: typecheck
		writer.On("SetWriteDeadline", expectedDeadline).Return((error)(nil)).Once()
		// nolint: typecheck
		writer.On("WritePreparedMessage", mock.MatchedBy(func(*websocket.PreparedMessage) bool { return true })).Return(expectedError).Once()

		assert.Equal(expectedError, pinger())
		assert.Zero(counter.Value())

		// nolint: typecheck
		writer.AssertExpectations(t)
	})
}

func TestInstrumentReader(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)

			statistics         = NewStatistics(nil, time.Now())
			reader             = new(mockConnectionReader)
			expectedData       = []byte{1, 2, 3, 4, 5, 6}
			instrumentedReader = InstrumentReader(reader, statistics)
		)

		require.NotNil(instrumentedReader)
		// nolint: typecheck
		reader.On("ReadMessage").Return(websocket.BinaryMessage, expectedData, (error)(nil)).Once()

		messageType, actualData, actualError := instrumentedReader.ReadMessage()
		assert.Equal(websocket.BinaryMessage, messageType)
		assert.Equal(expectedData, actualData)
		assert.NoError(actualError)
		assert.Equal(len(expectedData), statistics.BytesReceived())
		assert.Equal(1, statistics.MessagesReceived())

		// nolint: typecheck
		reader.AssertExpectations(t)
	})

	t.Run("ReadMessageError", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)

			statistics         = NewStatistics(nil, time.Now())
			reader             = new(mockConnectionReader)
			expectedError      = errors.New("expected")
			instrumentedReader = InstrumentReader(reader, statistics)
		)

		require.NotNil(instrumentedReader)
		// nolint: typecheck
		reader.On("ReadMessage").Return(-1, []byte{}, expectedError).Once()

		messageType, actualData, actualError := instrumentedReader.ReadMessage()
		assert.Equal(-1, messageType)
		assert.Len(actualData, 0)
		assert.Equal(expectedError, actualError)
		assert.Zero(statistics.BytesReceived())
		assert.Zero(statistics.MessagesReceived())

		// nolint: typecheck
		reader.AssertExpectations(t)
	})
}

func TestInstrumentWriter(t *testing.T) {
	t.Run("WriteMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				statistics         = NewStatistics(nil, time.Now())
				writer             = new(mockConnectionWriter)
				expectedData       = []byte{43, 3, 74, 111, 89}
				instrumentedWriter = InstrumentWriter(writer, statistics)
			)

			require.NotNil(instrumentedWriter)
			// nolint: typecheck
			writer.On("WriteMessage", websocket.BinaryMessage, expectedData).Return((error)(nil)).Once()

			assert.NoError(instrumentedWriter.WriteMessage(websocket.BinaryMessage, expectedData))
			assert.Equal(len(expectedData), statistics.BytesSent())
			assert.Equal(1, statistics.MessagesSent())

			// nolint: typecheck
			writer.AssertExpectations(t)
		})

		t.Run("Error", func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				statistics         = NewStatistics(nil, time.Now())
				writer             = new(mockConnectionWriter)
				expectedData       = []byte{128, 76, 14, 9, 178, 2, 126, 23}
				expectedError      = errors.New("expected")
				instrumentedWriter = InstrumentWriter(writer, statistics)
			)

			require.NotNil(instrumentedWriter)
			// nolint: typecheck
			writer.On("WriteMessage", websocket.BinaryMessage, expectedData).Return(expectedError).Once()

			assert.Equal(expectedError, instrumentedWriter.WriteMessage(websocket.BinaryMessage, expectedData))
			assert.Zero(statistics.BytesSent())
			assert.Zero(statistics.MessagesSent())

			// nolint: typecheck
			writer.AssertExpectations(t)
		})
	})

	t.Run("WritePreparedMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				statistics         = NewStatistics(nil, time.Now())
				writer             = new(mockConnectionWriter)
				instrumentedWriter = InstrumentWriter(writer, statistics)
			)

			require.NotNil(instrumentedWriter)

			expectedMessage, err := websocket.NewPreparedMessage(websocket.BinaryMessage, []byte{99, 44, 55, 128, 6, 2})
			require.NoError(err)
			require.NotNil(expectedMessage)

			// nolint: typecheck
			writer.On("WritePreparedMessage", expectedMessage).Return((error)(nil)).Once()

			assert.NoError(instrumentedWriter.WritePreparedMessage(expectedMessage))
			assert.Zero(statistics.BytesSent())
			assert.Equal(1, statistics.MessagesSent())

			// nolint: typecheck
			writer.AssertExpectations(t)
		})

		t.Run("Error", func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				statistics         = NewStatistics(nil, time.Now())
				writer             = new(mockConnectionWriter)
				expectedError      = errors.New("expected")
				instrumentedWriter = InstrumentWriter(writer, statistics)
			)

			require.NotNil(instrumentedWriter)

			expectedMessage, err := websocket.NewPreparedMessage(websocket.BinaryMessage, []byte{8, 45, 123, 79, 1})
			require.NoError(err)
			require.NotNil(expectedMessage)

			// nolint: typecheck
			writer.On("WritePreparedMessage", expectedMessage).Return(expectedError).Once()

			assert.Equal(expectedError, instrumentedWriter.WritePreparedMessage(expectedMessage))
			assert.Zero(statistics.BytesSent())
			assert.Zero(statistics.MessagesSent())

			// nolint: typecheck
			writer.AssertExpectations(t)
		})
	})
}
