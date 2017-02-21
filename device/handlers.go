package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

type Failures map[Interface]error

func (df Failures) Add(d Interface, deviceError error) {
	df[d] = deviceError
}

func (df Failures) MarshalJSON() ([]byte, error) {
	var (
		buffer    = bytes.NewBufferString(`{"errors": [`)
		separator = ""
	)

	for d, deviceError := range df {
		if deviceError != nil {
			fmt.Fprintf(buffer, `{"id": "%s", "key": "%s", error: "%s"}%s`, d.ID(), d.Key(), deviceError, separator)
			separator = ","
		}
	}

	buffer.WriteString(`]}`)
	return buffer.Bytes(), nil
}

func (df Failures) WriteResponse(response http.ResponseWriter) error {
	if len(df) > 0 {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusInternalServerError)
		data, _ := df.MarshalJSON()
		_, err := response.Write(data)
		return err
	}

	response.WriteHeader(http.StatusOK)
	return nil
}

func NewJSONHandler(decoder *wrp.DecoderPool, router Router) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		message, err := decoder.DecodeMessage(request.Body)
		if err != nil {
			http.Error(
				response,
				fmt.Sprintf("Could not decode WRP message: %s", err),
				http.StatusBadRequest,
			)

			return
		}

		var (
			failures                  = make(Failures)
			_, totalCount, routeError = router.Route(message, failures.Add)
		)

		if routeError != nil {
			http.Error(
				response,
				fmt.Sprintf("Unable to route message: %s", routeError),
				http.StatusBadRequest,
			)
		} else if totalCount == 0 {
			response.WriteHeader(http.StatusNotFound)
		}

		failures.WriteResponse(response)
	})
}

func NewMsgpackHandler(decoder *wrp.DecoderPool, router Router) http.Handler {
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

		message, err := decoder.DecodeMessageBytes(body)
		if err != nil {
			http.Error(
				response,
				fmt.Sprintf("Could not decode WRP message: %s", err),
				http.StatusBadRequest,
			)

			return
		}

		var (
			failures                  = make(Failures)
			_, totalCount, routeError = router.RouteUsing(message, body, failures.Add)
		)

		if routeError != nil {
			http.Error(
				response,
				fmt.Sprintf("Unable to route message: %s", routeError),
				http.StatusBadRequest,
			)
		} else if totalCount == 0 {
			response.WriteHeader(http.StatusNotFound)
		}

		failures.WriteResponse(response)
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
