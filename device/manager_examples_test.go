package device

import (
	"bytes"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"io"
	"os"
	"sync"
)

func ExampleManagerTransaction() {
	var (
		// we use a WaitGroup so that we can predictably order output
		messageReceived = new(sync.WaitGroup)

		options = &Options{
			Logger: logging.DefaultLogger(),
			Listeners: []Listener{
				func(e *Event) {
					switch e.Type {
					case Connect:
						fmt.Printf("%s connected\n", e.Device.ID())
						messageReceived.Add(1)
					case MessageReceived:
						fmt.Println("response received")
						messageReceived.Done()
					}
				},
			},
		}

		manager, server, websocketURL = startWebsocketServer(options)

		dialer                   = NewDialer(options, nil)
		connection, _, dialError = dialer.Dial(
			websocketURL,
			"mac:111122223333",
			nil,
			nil,
		)
	)

	defer server.Close()
	if dialError != nil {
		fmt.Fprintf(os.Stderr, "Dial error: %s\n", dialError)
		return
	}

	defer connection.Close()

	// spawn a goroutine that simply waits for a request/response and responds
	go func() {
		readFrame := new(bytes.Buffer)
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
				// casting is fine here since we know what type message is
				response = message.Response("mac:111122223333", 1).(*wrp.Message)
				encoder  = wrp.NewEncoder(writeFrame, wrp.Msgpack)
			)

			response.Payload = []byte("Homer Simpson, Smiling Politely")
			writeError = encoder.Encode(response)
		}

		if writeError != nil {
			fmt.Fprintf(os.Stderr, "Could not write response: %s\n", writeError)
		}
	}()

	// Route will block until the corresponding message returns from the device
	response, err := manager.Route(
		&Request{
			Message: &wrp.SimpleRequestResponse{
				Source:          "Example",
				Destination:     "mac:111122223333",
				Payload:         []byte("Billy Corgan, Smashing Pumpkins"),
				TransactionUUID: "MyTransactionUUID",
			},
		},
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Route error: %s\n", err)
	} else if response != nil {
		messageReceived.Wait()
		fmt.Printf("(%s): %s to %s -> %s\n", response.Message.TransactionUUID, response.Message.Source, response.Message.Destination, response.Message.Payload)
	}

	// Output:
	// mac:111122223333 connected
	// response received
	// (MyTransactionUUID): mac:111122223333 to Example -> Homer Simpson, Smiling Politely
}
