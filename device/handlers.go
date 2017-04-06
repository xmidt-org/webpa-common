package device

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"io"
	"net/http"
	"sync"
	"time"
)

const DefaultMessageTimeout time.Duration = time.Minute * 2

// MessageHandler is a configurable http.Handler which handles inbound WRP traffic
// to be sent to devices.
type MessageHandler struct {
	// Logger is the sink for logging output.  If not set, logging will be sent to logging.DefaultLogger().
	Logger logging.Logger

	// Decoders is the pool of wrp.Decoder objects used to decode http.Request bodies
	// sent to this handler.  This field is required.
	Decoders *wrp.DecoderPool

	// Encoders is the optional pool of wrp.Encoder objects used to encode wrp messages sent
	// as HTTP responses.  If not supplied, this handler assumes the format returned by the Router
	// is the format to be sent back in the HTTP response.
	Encoders *wrp.EncoderPool

	// Router is the device message Router to use.  This field is required.
	Router Router

	// Timeout is the optional timeout for all operations through this handler.
	// If this field is unset or is nonpositive, DefaultMessageTimeout is used instead.
	Timeout time.Duration
}

func (mh *MessageHandler) logger() logging.Logger {
	if mh.Logger != nil {
		return mh.Logger
	}

	return logging.DefaultLogger()
}

// createContext creates the Context object for routing operations.
// This method will never return nils.  There will always be a timeout on the
// returned context, which means there will always be a cancel function too.
func (mh *MessageHandler) createContext(httpRequest *http.Request) (context.Context, context.CancelFunc) {
	timeout := mh.Timeout
	if timeout < 1 {
		timeout = DefaultMessageTimeout
	}

	return context.WithTimeout(httpRequest.Context(), mh.Timeout)
}

// decodeRequest transforms an HTTP request into a device request.
func (mh *MessageHandler) decodeRequest(ctx context.Context, httpRequest *http.Request) (deviceRequest *Request, err error) {
	deviceRequest, err = DecodeRequest(httpRequest.Body, mh.Decoders)
	if err == nil {
		deviceRequest = deviceRequest.WithContext(ctx)
	}

	return
}

func (mh *MessageHandler) ServeHTTP(httpResponse http.ResponseWriter, httpRequest *http.Request) {
	ctx, cancel := mh.createContext(httpRequest)
	defer cancel()

	deviceRequest, err := mh.decodeRequest(ctx, httpRequest)
	if err != nil {
		httperror.Formatf(
			httpResponse,
			http.StatusBadRequest,
			"Could not decode WRP message: %s",
			err,
		)

		return
	}

	// deviceRequest carries the context through the routing infrastructure
	if deviceResponse, err := mh.Router.Route(deviceRequest); err != nil {
		code := http.StatusInternalServerError
		switch err {
		case ErrorInvalidDeviceName:
			code = http.StatusBadRequest
		case ErrorDeviceNotFound:
			code = http.StatusNotFound
		case ErrorNonUniqueID:
			code = http.StatusBadRequest
		case ErrorInvalidTransactionKey:
			code = http.StatusBadRequest
		case ErrorTransactionAlreadyRegistered:
			code = http.StatusBadRequest
		}

		httperror.Formatf(
			httpResponse,
			code,
			"Could not process device request: %s",
			err,
		)
	} else if deviceResponse != nil {
		if err := EncodeResponse(httpResponse, deviceResponse, mh.Encoders); err != nil {
			mh.logger().Error("Error while writing transaction response: %s", err)
		}
	}

	// if deviceReponse == nil, that just means the request was not something that represented
	// the start of a transaction.  For example, events do not carry a transaction key because
	// they do not expect responses.
}

type ConnectHandler struct {
	Logger         logging.Logger
	Connector      Connector
	ResponseHeader http.Header
}

func (ch *ConnectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	logger := ch.Logger
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	device, err := ch.Connector.Connect(response, request, ch.ResponseHeader)
	if err != nil {
		logger.Error("Failed to connect device: %s", err)
	} else {
		logger.Debug("Connected device: %s", device.ID())
	}
}

// ListHandler is an http.Handler that responds with lists of devices associated with a Registry.
type ListHandler struct {
	Logger logging.Logger

	// Registry is the required instance containing registered devices
	Registry Registry

	// CachePeriod is the length of time that the internally cached JSON is considered valid.
	// If nonpositive, this handler does no caching.
	CachePeriod time.Duration

	lock        sync.RWMutex
	cacheExpiry time.Time
	cachedJSON  []byte
}

func (lh *ListHandler) logger() logging.Logger {
	if lh.Logger != nil {
		return lh.Logger
	}

	return logging.DefaultLogger()
}

func (lh *ListHandler) generateList(output io.Writer) (err error) {
	_, err = fmt.Fprint(output, `{"devices":[`)
	if err != nil {
		return
	}

	comma := ""
	lh.Registry.VisitAll(func(device Interface) {
		if err == nil {
			_, err = fmt.Fprint(output, comma, device.String())
			comma = ","
		}
	})

	if err == nil {
		_, err = fmt.Fprint(output, `]}`)
	}

	return
}

func (lh *ListHandler) tryCache() (json []byte, expired bool) {
	lh.lock.RLock()
	defer lh.lock.RUnlock()

	expired = lh.cacheExpiry.Before(time.Now())
	json = lh.cachedJSON

	return
}

func (lh *ListHandler) updateCache() (json []byte, err error) {
	lh.lock.Lock()
	defer lh.lock.Unlock()

	if lh.cacheExpiry.Before(time.Now()) {
		json = lh.cachedJSON
		return
	}

	var output bytes.Buffer
	err = lh.generateList(&output)
	if err != nil {
		return
	}

	lh.cacheExpiry = time.Now().Add(lh.CachePeriod)
	lh.cachedJSON = output.Bytes()
	json = lh.cachedJSON
	return
}

func (lh *ListHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if lh.CachePeriod < 1 {
		// uncached
		response.Header().Set("Content-Type", "application/json")
		if err := lh.generateList(response); err != nil {
			lh.logger().Error("Unable to output device list: %s", err)
		}

		return
	}

	json, expired := lh.tryCache()
	if expired {
		var err error
		if json, err = lh.updateCache(); err != nil {
			httperror.Formatf(
				response,
				http.StatusInternalServerError,
				"Could not update cached device list: %s",
				err,
			)

			return
		}
	}

	response.Header().Set("Content-Type", "application/json")
	if _, err := response.Write(json); err != nil {
		lh.logger().Error("Unable to output device list: %s", err)
	}
}
