package device

import (
	"encoding/json"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

func NewJSONHandler(router Router) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var (
			decoder = wrp.NewDecoder(request.Body, wrp.JSON)
			message = new(wrp.Message)
		)

		if err := decoder.Decode(message); err != nil {
			http.Error(
				response,
				fmt.Sprintf("Could not decode WRP message: %s", err),
				http.StatusBadRequest,
			)

			return
		}

		if _, _, err := router.Route(message, nil); err != nil {
			http.Error(
				response,
				fmt.Sprintf("Unable to route message: %s", err),
				http.StatusBadRequest,
			)

			return
		}

		response.WriteHeader(http.StatusAccepted)
	})
}

func NewMsgpackHandler(router Router) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			http.Error(
				response,
				fmt.Sprintf("Unable to read request body: %s", err),
				http.StatusBadRequest,
			)

			return
		}

		var (
			decoder = wrp.NewDecoderBytes(body, wrp.Msgpack)
			message = new(wrp.Message)
		)

		if err := decoder.Decode(message); err != nil {
			http.Error(
				response,
				fmt.Sprintf("Could not decode WRP message: %s", err),
				http.StatusBadRequest,
			)

			return
		}

		if _, _, err := router.RouteUsing(message, body, nil); err != nil {
			http.Error(
				response,
				fmt.Sprintf("Unable to route message: %s", err),
				http.StatusBadRequest,
			)

			return
		}

		response.WriteHeader(http.StatusAccepted)
	})
}

// NewConnectHandler produces an http.Handler that allows devices to connect
// to a specific Manager.
func NewConnectHandler(connector Connector, responseHeader http.Header, logger logging.Logger) http.Handler {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		device, err := connector.Connect(response, request, responseHeader)
		if err != nil {
			logger.Error("Failed to connect device: %s", err)
		} else {
			logger.Debug("Connected device: %s", device.ID())
		}
	})
}

// NewDeviceListHandler returns an http.Handler that renders a JSON listing
// of the devices within a manager.
func NewDeviceListHandler(manager Manager, logger logging.Logger) http.Handler {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		flusher := response.(http.Flusher)
		response.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(response, `{"device": [`); err != nil {
			logger.Error("Unable to write content: %s", err)
			return
		}

		devices := make(chan Interface, 100)
		finish := new(sync.WaitGroup)
		finish.Add(1)

		// to minimize the time we hold the read lock on the Manager, spawn a goroutine
		// that collects devices and inserts them into an output buffer
		go func() {
			defer finish.Done()

			needsDelimiter := false
			for d := range devices {
				if needsDelimiter {
					io.WriteString(response, ",")
				}

				needsDelimiter = true
				if data, err := json.Marshal(d); err != nil {
					message := fmt.Sprintf("Unable to marshal device [%s] as JSON: %s", d.ID(), err)
					logger.Error(message)
					fmt.Fprintf(response, `"%s"`, message)
				} else {
					response.Write(data)
				}

				flusher.Flush()
			}
		}()

		manager.VisitAll(func(d Interface) {
			devices <- d
		})

		close(devices)
		finish.Wait()
		io.WriteString(response, `]}`)
		flusher.Flush()
	})
}
