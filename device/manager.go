// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xmidt-org/webpa-common/v2/convey"
	"github.com/xmidt-org/webpa-common/v2/convey/conveymetric"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/gorilla/websocket"
	"github.com/xmidt-org/webpa-common/v2/convey/conveyhttp"
	"github.com/xmidt-org/webpa-common/v2/xhttp"
	"github.com/xmidt-org/wrp-go/v3"
)

const MaxDevicesHeader = "X-Xmidt-Max-Devices"

// DefaultWRPContentType is the content type used on inbound WRP messages which don't provide one.
const DefaultWRPContentType = "application/octet-stream"

// WRPTimestampMetadataKey is the uniform timestamp given to all device wrp messsages (expect for message sent to devices `writePump`)
const WRPTimestampMetadataKey = "/xmidt-timestamp"

// emptyBuffer is solely used as an address of a global empty buffer.
// This sentinel value will reset pointers of the writePump's encoder
// such that the gc can clean things up.
var emptyBuffer = []byte{}

// Connector is a strategy interface for managing device connections to a server.
// Implementations are responsible for upgrading websocket connections and providing
// for explicit disconnection.
type Connector interface {
	// Connect upgrade an HTTP connection to a websocket and begins concurrent
	// management of the device.
	Connect(http.ResponseWriter, *http.Request, http.Header) (Interface, error)

	// Disconnect disconnects the device associated with the given id.
	// If the id was found, this method returns true.
	Disconnect(ID, CloseReason) bool

	// DisconnectIf iterates over all devices known to this manager, applying the
	// given predicate.  For any devices that result in true, this method disconnects them.
	// Note that this method may pause connections and disconnections while it is executing.
	// This method returns the number of devices that were disconnected.
	//
	// Only disconnection by ID is supported, which means that any identifier matching
	// the predicate will result in *all* duplicate devices under that ID being removed.
	//
	// No methods on this Manager should be called from within the predicate function, or
	// a deadlock will likely occur.
	DisconnectIf(func(ID) (CloseReason, bool)) int

	// DisconnectAll disconnects all devices from this instance, and returns the count of
	// devices disconnected.
	DisconnectAll(CloseReason) int

	// GetFilter returns the Filter interface used for filtering connection requests
	GetFilter() Filter
}

// Router handles dispatching messages to devices.
type Router interface {
	// Route dispatches a WRP request to exactly one device, identified by the ID
	// field of the request.  Route is synchronous, and honors the cancellation semantics
	// of the Request's context.
	Route(*Request) (*Response, error)
}

// Registry is the strategy interface for querying the set of connected devices.  Methods
// in this interface follow the Visitor pattern and are typically executed under a read lock.
type Registry interface {
	// Len returns the count of devices currently in this registry
	Len() int

	// Get returns the device associated with the given ID, if any
	Get(ID) (Interface, bool)

	// VisitAll applies the given visitor function to each device known to this manager.
	//
	// No methods on this Manager should be called from within the visitor function, or
	// a deadlock will likely occur.
	VisitAll(func(Interface) bool) int
}

type Filter interface {
	AllowConnection(d Interface) (bool, MatchResult)
}

type MatchResult struct {
	Location string
	Key      string
}

type FilterFunc func(d Interface) (bool, MatchResult)

func (filter FilterFunc) AllowConnection(d Interface) (bool, MatchResult) {
	return filter(d)
}

// Manager supplies a hub for connecting and disconnecting devices as well as
// an access point for obtaining device metadata.
type Manager interface {
	Connector
	Router
	Registry
	MaxDevices() int
}

// ManagerOption is a configuration option for a manager
type ManagerOption func(*manager)

// NewManager constructs a Manager from a set of options.  A ConnectionFactory will be
// created from the options if one is not supplied.
func NewManager(o *Options) Manager {
	var (
		logger   = o.logger()
		measures = NewMeasures(o.metricsProvider())
		wrpCheck = o.wrpCheck()
	)

	logger.Debug("source check configuration", zap.String("type", string(wrpCheck.Type)))

	return &manager{
		logger:           logger,
		readDeadline:     NewDeadline(o.idlePeriod(), o.now()),
		writeDeadline:    NewDeadline(o.writeTimeout(), o.now()),
		upgrader:         o.upgrader(),
		conveyTranslator: conveyhttp.NewHeaderTranslator("", nil),
		devices: newRegistry(registryOptions{
			Logger:   logger,
			Limit:    o.maxDevices(),
			Measures: measures,
		}),
		conveyHWMetric: conveymetric.NewConveyMetric(measures.Models, []conveymetric.TagLabelPair{
			{
				Tag:   "hw-model",
				Label: "model",
			},
			{
				Tag:   "fw-name",
				Label: "firmware",
			}}...),

		deviceMessageQueueSize: o.deviceMessageQueueSize(),
		pingPeriod:             o.pingPeriod(),

		listeners:             o.listeners(),
		measures:              measures,
		enforceWRPSourceCheck: wrpCheck.Type == CheckTypeEnforce,
		filter:                o.filter(),
	}
}

// manager is the internal Manager implementation.
type manager struct {
	logger *zap.Logger

	readDeadline     func() time.Time
	writeDeadline    func() time.Time
	upgrader         *websocket.Upgrader
	conveyTranslator conveyhttp.HeaderTranslator

	devices        *registry
	conveyHWMetric conveymetric.Interface

	deviceMessageQueueSize int
	pingPeriod             time.Duration

	listeners             []Listener
	measures              Measures
	enforceWRPSourceCheck bool

	filter Filter
}

func (m *manager) Connect(response http.ResponseWriter, request *http.Request, responseHeader http.Header) (Interface, error) {
	m.logger.Debug("device connect", zap.Any("url", request.URL))
	ctx := request.Context()
	id, ok := GetID(ctx)
	if !ok {
		xhttp.WriteError(
			response,
			http.StatusInternalServerError,
			ErrorMissingDeviceNameContext,
		)

		return nil, ErrorMissingDeviceNameContext
	}

	metadata, ok := GetDeviceMetadata(ctx)
	if !ok {
		metadata = new(Metadata)
	}

	cvy, cvyErr := m.conveyTranslator.FromHeader(request.Header)
	d := newDevice(deviceOptions{
		ID:         id,
		C:          cvy,
		Compliance: convey.GetCompliance(cvyErr),
		QueueSize:  m.deviceMessageQueueSize,
		Metadata:   metadata,
		Logger:     m.logger,
	})

	if allow, matchResults := m.filter.AllowConnection(d); !allow {
		d.logger.Info("filter match found", zap.String("location", matchResults.Location), zap.String("key", matchResults.Key))
		return nil, ErrorDeviceFilteredOut
	}

	if len(metadata.Claims()) < 1 {
		d.logger.Error("missing security information")
	}

	if cvyErr == nil {
		d.logger.Info(fmt.Sprintf("convey: %v", cvy))
	} else {
		d.logger.Error("bad or missing convey data", zap.Error(cvyErr))
	}

	c, err := m.upgrader.Upgrade(response, request, responseHeader)
	if err != nil {
		d.logger.Error("failed websocket upgrade", zap.Error(err))
		return nil, err
	}

	d.logger.Debug("websocket upgrade complete", zap.String("localAddress", c.LocalAddr().String()))

	pinger, err := NewPinger(c, m.measures.Ping, []byte(d.ID()), m.writeDeadline)
	if err != nil {
		d.logger.Error("unable to create pinger", zap.Error(err))
		c.Close()
		return nil, err
	}

	if err := m.devices.add(d); err != nil {
		d.logger.Error("unable to register device", zap.Error(err))
		c.Close()
		return nil, err
	}

	event := &Event{
		Type:   Connect,
		Device: d,
	}

	if cvyErr == nil {
		bytes, err := json.Marshal(cvy)
		if err == nil {
			// nolint: typecheck
			event.Format = wrp.JSON
			event.Contents = bytes
		} else {
			d.logger.Error("unable to marshal the convey header", zap.Error(err))
		}
	}
	metricClosure, err := m.conveyHWMetric.Update(cvy, "partnerid", metadata.PartnerIDClaim(), "trust", strconv.Itoa(metadata.TrustClaim()))
	if err != nil {
		d.logger.Error("failed to update convey metrics", zap.Error(err))
	}

	d.conveyClosure = metricClosure
	m.dispatch(event)

	SetPongHandler(c, m.measures.Pong, m.readDeadline)
	closeOnce := new(sync.Once)
	go m.readPump(d, InstrumentReader(c, d.statistics), closeOnce)
	go m.writePump(d, InstrumentWriter(c, d.statistics), pinger, closeOnce)

	d.logger.Debug("Connection metadata", zap.String("conveyCompliance", convey.GetCompliance(cvyErr).String()), zap.Strings("conveyHeaderKeys", maps.Keys(cvy)), zap.Any("conveyHeader", cvy))

	return d, nil
}

func (m *manager) dispatch(e *Event) {
	for _, listener := range m.listeners {
		listener(e)
	}
}

// pumpClose handles the proper shutdown and logging of a device's pumps.
// This method should be executed within a sync.Once, so that it only executes
// once for a given device.
//
// Note that the write pump does additional cleanup.  In particular, the write pump
// dispatches message failed events for any messages that were waiting to be delivered
// at the time of pump closure.
func (m *manager) pumpClose(d *device, c io.Closer, reason CloseReason) {

	if !m.isDeviceDuplicated(d) {
		// remove will invoke requestClose()
		m.devices.remove(d.id, reason)
	}

	closeError := c.Close()

	d.logger.Error("Closed device connection",
		zap.NamedError("closeError", closeError), zap.String("reasonError", reason.String()), zap.String("reason", reason.Text),
		// nolint: typecheck
		zap.String("finalStatistics", d.Statistics().String()))

	m.dispatch(
		&Event{
			Type:   Disconnect,
			Device: d,
		},
	)
	d.conveyClosure()
}

// nolint: typecheck
func (m *manager) wrpSourceIsValid(message *wrp.Message, d *device) bool {
	expectedID := d.ID()
	if len(strings.TrimSpace(message.Source)) == 0 {
		d.logger.Error("WRP source was empty", zap.Int("trustLevel", d.Metadata().TrustClaim()))
		if m.enforceWRPSourceCheck {
			m.measures.WRPSourceCheck.With("outcome", "rejected", "reason", "empty").Add(1)
			return false
		}
		m.measures.WRPSourceCheck.With("outcome", "accepted", "reason", "empty").Add(1)
		return true
	}

	actualID, err := ParseID(message.Source)
	if err != nil {
		d.logger.Error("Failed to parse ID from WRP source", zap.Int("trustLevel", d.Metadata().TrustClaim()))
		if m.enforceWRPSourceCheck {
			m.measures.WRPSourceCheck.With("outcome", "rejected", "reason", "parse_error").Add(1)
			return false
		}
		m.measures.WRPSourceCheck.With("outcome", "accepted", "reason", "parse_error").Add(1)
		return true
	}

	if expectedID != actualID {
		d.logger.Error("ID in WRP source does not match device's ID", zap.String("spoofedID", string(actualID)), zap.Int("trustLevel", d.Metadata().TrustClaim()))
		if m.enforceWRPSourceCheck {
			m.measures.WRPSourceCheck.With("outcome", "rejected", "reason", "id_mismatch").Add(1)
			return false
		}
		m.measures.WRPSourceCheck.With("outcome", "accepted", "reason", "id_mismatch").Add(1)
		return true
	}

	m.measures.WRPSourceCheck.With("outcome", "accepted", "reason", "id_match").Add(1)
	return true
}

// nolint: typecheck
func addDeviceMetadataContext(message *wrp.Message, deviceMetadata *Metadata) {
	if message.Metadata == nil {
		message.Metadata = make(map[string]string)
	}

	message.Metadata[WRPTimestampMetadataKey] = time.Now().Format(time.RFC3339Nano)
	message.PartnerIDs = []string{deviceMetadata.PartnerIDClaim()}

	// nolint: typecheck
	if message.Type == wrp.SimpleEventMessageType {
		message.SessionID = deviceMetadata.SessionID()
	}
}

// readPump is the goroutine which handles the stream of WRP messages from a device.
// This goroutine exits when any error occurs on the connection.
func (m *manager) readPump(d *device, r ReadCloser, closeOnce *sync.Once) {
	defer d.logger.Debug("readPump exiting")
	d.logger.Debug("readPump starting")

	var (
		readError error
		// nolint: typecheck
		decoder = wrp.NewDecoder(nil, wrp.Msgpack)
		// nolint: typecheck
		encoder = wrp.NewEncoder(nil, wrp.Msgpack)
	)

	// all the read pump has to do is ensure the device and the connection are closed
	// it is the write pump's responsibility to do further cleanup
	defer func() {
		closeOnce.Do(func() { m.pumpClose(d, r, CloseReason{Err: readError, Text: "readerror"}) })
	}()

	for {
		messageType, data, readError := r.ReadMessage()
		if readError != nil {
			d.logger.Error("read error", zap.Error(readError))
			return
		}

		if messageType != websocket.BinaryMessage {
			d.logger.Error("skipping non-binary frame", zap.Int("messageType", messageType))
			continue
		}

		var (
			// nolint: typecheck
			message = new(wrp.Message)
			event   = Event{
				Type:    MessageReceived,
				Device:  d,
				Message: message,
				// nolint: typecheck
				Format:   wrp.Msgpack,
				Contents: data,
			}
		)

		decoder.ResetBytes(data)
		err := decoder.Decode(message)
		if err != nil {
			d.logger.Error("skipping malformed WRP message", zap.Error(err))
			continue
		}

		// nolint: typecheck
		err = wrp.UTF8(message)
		if err != nil {
			d.logger.Error("skipping malformed WRP message", zap.Error(err))
			continue
		}

		if !m.wrpSourceIsValid(message, d) {
			d.logger.Error("skipping WRP message with invalid source")
			continue
		}

		if len(strings.TrimSpace(message.ContentType)) == 0 {
			message.ContentType = DefaultWRPContentType
		}

		addDeviceMetadataContext(message, d.Metadata())

		// nolint: typecheck
		if message.Type == wrp.SimpleRequestResponseMessageType {
			m.measures.RequestResponse.Add(1.0)
		}

		encoder.ResetBytes(&event.Contents)
		err = encoder.Encode(message)

		if err != nil {
			d.logger.Error("unable to encode WRP message", zap.Error(err))
			continue
		}

		// update any waiting transaction
		if message.IsTransactionPart() {
			err := d.transactions.Complete(
				message.TransactionKey(),
				&Response{
					Device:  d,
					Message: message,
					// nolint: typecheck
					Format:   wrp.Msgpack,
					Contents: event.Contents,
				},
			)

			if err != nil {
				d.logger.Error("Error while completing transaction", zap.Error(err), zap.String("transactionKey", message.TransactionKey()))
				event.Type = TransactionBroken
				event.Error = err
			} else {
				event.Type = TransactionComplete
			}
		}
		m.dispatch(&event)
	}
}

// writePump is the goroutine which services messages addressed to the device.
// this goroutine exits when either an explicit shutdown is requested or any
// error occurs on the connection.
func (m *manager) writePump(d *device, w WriteCloser, pinger func() error, closeOnce *sync.Once) {
	defer d.logger.Debug("writePump exiting")
	d.logger.Debug("writePump starting")

	var (
		envelope *envelope
		// nolint: typecheck
		encoder    = wrp.NewEncoder(nil, wrp.Msgpack)
		writeError error

		pingTicker = time.NewTicker(m.pingPeriod)
	)

	// cleanup: we not only ensure that the device and connection are closed but also
	// ensure that any messages that were waiting and/or failed are dispatched to
	// the configured listener
	defer func() {
		pingTicker.Stop()
		closeOnce.Do(func() { m.pumpClose(d, w, CloseReason{Err: writeError, Text: "write-error"}) })

		// notify listener of any message that just now failed
		// any writeError is passed via this event
		if envelope != nil {
			m.dispatch(&Event{
				Type:     MessageFailed,
				Device:   d,
				Message:  envelope.request.Message,
				Format:   envelope.request.Format,
				Contents: envelope.request.Contents,
				Error:    writeError,
			})
		}

		// drain the messages, dispatching them as message failed events.  we never close
		// the message channel, so just drain until a receive would block.
		//
		// Nil is passed explicitly as the error to indicate that these messages failed due
		// to the device disconnecting, not due to an actual I/O error.
		for {
			select {
			case undeliverable := <-d.messages:
				d.logger.Error("undeliverable message", zap.Any("deviceMessage", undeliverable))
				m.dispatch(&Event{
					Type:     MessageFailed,
					Device:   d,
					Message:  undeliverable.request.Message,
					Format:   undeliverable.request.Format,
					Contents: undeliverable.request.Contents,
					Error:    writeError,
				})
			default:
				return
			}
		}
	}()

	for writeError == nil {
		envelope = nil

		select {
		case <-d.shutdown:
			d.logger.Debug("explicit shutdown")
			// nolint: typecheck
			writeError = w.Close()
			return

		case envelope = <-d.messages:
			var frameContents []byte
			// nolint: typecheck
			if envelope.request.Format == wrp.Msgpack && len(envelope.request.Contents) > 0 {
				frameContents = envelope.request.Contents
			} else {
				// if the request was in a format other than Msgpack, or if the caller did not pass
				// Contents, then do the encoding here.
				encoder.ResetBytes(&frameContents)
				writeError = encoder.Encode(envelope.request.Message)
				encoder.ResetBytes(&emptyBuffer)
			}

			if writeError == nil {
				writeError = w.WriteMessage(websocket.BinaryMessage, frameContents)
			}

			event := Event{
				Device:   d,
				Message:  envelope.request.Message,
				Format:   envelope.request.Format,
				Contents: envelope.request.Contents,
				Error:    writeError,
			}

			if writeError != nil {
				envelope.complete <- writeError
				event.Type = MessageFailed
			} else {
				event.Type = MessageSent
			}

			close(envelope.complete)
			m.dispatch(&event)

		case <-pingTicker.C:
			writeError = pinger()
		}
	}
}

func (m *manager) Disconnect(id ID, reason CloseReason) bool {
	_, ok := m.devices.remove(id, reason)
	return ok
}

func (m *manager) DisconnectIf(filter func(ID) (CloseReason, bool)) int {
	return m.devices.removeIf(func(d *device) (CloseReason, bool) {
		return filter(d.id)
	})
}

func (m *manager) DisconnectAll(reason CloseReason) int {
	return m.devices.removeAll(reason)
}

func (m *manager) GetFilter() Filter {
	return m.filter
}

func defaultFilterFunc() FilterFunc {
	return func(d Interface) (bool, MatchResult) {
		return true, MatchResult{}
	}
}

func (m *manager) Len() int {
	return m.devices.len()
}

func (m *manager) Get(id ID) (Interface, bool) {
	return m.devices.get(id)
}

func (m *manager) VisitAll(visitor func(Interface) bool) int {
	return m.devices.visit(func(d *device) bool {
		return visitor(d)
	})
}

func (m *manager) Route(request *Request) (*Response, error) {
	if destination, err := request.ID(); err != nil {
		return nil, err
	} else if d, ok := m.devices.get(destination); ok {
		return d.Send(request)
	} else {
		return nil, ErrorDeviceNotFound
	}
}

func (m *manager) MaxDevices() int {
	return m.devices.limit
}

func (m *manager) isDeviceDuplicated(d *device) bool {
	existing, ok := m.devices.get(d.id)
	if !ok {
		return false
	}
	return existing.state != d.state
}
