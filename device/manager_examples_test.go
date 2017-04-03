package device

import (
	"bytes"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"io"
	"io/ioutil"
	"os"
	"sync"
)

func ExampleManagerSimple() {
	var (
		options = &Options{
			Logger: &logging.LoggerWriter{ioutil.Discard},
			Listeners: []Listener{
				func(e *Event) {
					if e.Type == MessageReceived {
						message := e.Message.(*wrp.Message)
						fmt.Printf("%s to %s -> %s\n", message.Source, message.Destination, message.Payload)
					}
				},
			},
		}

		manager, server, websocketURL = startWebsocketServer(options)

		deviceDone = new(sync.WaitGroup)
	)

	defer server.Close()
	deviceDone.Add(1)

	// spawn a goroutine that simply waits for a request/response and responds
	go func() {
		defer deviceDone.Done()

		var (
			readFrame = new(bytes.Buffer)

			dialer             = NewDialer(options, nil)
			connection, _, err = dialer.Dial(
				websocketURL,
				"mac:111122223333",
				nil,
				nil,
			)
		)

		if err != nil {
			return
		}

		defer connection.Close()

		if frameRead, err := connection.Read(readFrame); !frameRead || err != nil {
			fmt.Fprintf(os.Stderr, "Read failed: frameRead=%b, err=%s\n", frameRead, err)
			return
		}

		var (
			message = new(wrp.Message)
			decoder = wrp.NewDecoder(readFrame, wrp.Msgpack)

			writeFrame io.WriteCloser
			writeError error
		)

		if err := decoder.Decode(message); err != nil {
			fmt.Fprintf(os.Stderr, "Could not decode message: %s\n", err)
			return
		}

		if writeFrame, writeError = connection.NextWriter(); writeError == nil {
			defer writeFrame.Close()
			var (
				response = message.Response("mac:111122223333", 1).(*wrp.Message)
				encoder  = wrp.NewEncoder(writeFrame, wrp.Msgpack)
			)

			response.Payload = []byte("Homer Simpson, Smiling Politely")
			fmt.Printf("%s to %s -> %s\n", response.Source, response.Destination, response.Payload)
			writeError = encoder.Encode(response)
		}

		if writeError != nil {
			fmt.Fprintf(os.Stderr, "Could not write response: %s\n", err)
		}
	}()

	response, err := manager.Route(
		&Request{
			Message: &wrp.SimpleRequestResponse{
				Source:      "Example",
				Destination: "mac:111122223333",
				Payload:     []byte("Billy Corgan, Smashing Pumpkins"),
			},
		},
	)

	deviceDone.Wait()

	// Output:
	// Example to mac:111122223333 -> Billy Corgan, Smashing Pumpkins
	// mac:111122223333 to Example -> Homer Simpson, smiling politely
}
