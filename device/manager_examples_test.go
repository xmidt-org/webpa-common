package device

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
)

func expectMessage(c Connection) (*wrp.Message, error) {
	var frame bytes.Buffer
	if ok, err := c.Read(&frame); !ok || err != nil {
		return nil, fmt.Errorf("Read failed: %s", err)
	}

	var (
		message = new(wrp.Message)
		decoder = wrp.NewDecoder(&frame, wrp.Msgpack)
	)

	if err := decoder.Decode(message); err != nil {
		return nil, fmt.Errorf("Could not decode message: %s", err)
	}

	return message, nil
}

func writeMessage(m *wrp.Message, c Connection) error {
	if frame, err := c.NextWriter(); err != nil {
		return err
	} else {
		encoder := wrp.NewEncoder(frame, wrp.Msgpack)
		if err := encoder.Encode(m); err != nil {
			return err
		}

		return frame.Close()
	}
}

func ExampleManagerTransaction() {
	var (
		transactionComplete = new(sync.WaitGroup)
		disconnected        = new(sync.WaitGroup)
	)

	transactionComplete.Add(1)
	disconnected.Add(1)

	var (
		options = &Options{
			Logger:    logging.DefaultLogger(),
			AuthDelay: 250 * time.Millisecond,
			Listeners: []Listener{
				func(e *Event) {
					switch e.Type {
					case Connect:
						fmt.Printf("%s connected\n", e.Device.ID())
					case MessageSent:
						if e.Message.MessageType() == wrp.AuthMessageType {
							fmt.Println("auth status sent")
						} else {
							fmt.Println("message sent")
						}
					case TransactionComplete:
						fmt.Println("response received")
						transactionComplete.Done()
					case Disconnect:
						fmt.Printf("%s disconnected\n", e.Device.ID())
						disconnected.Done()
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
		)
	)

	defer server.Close()
	if dialError != nil {
		fmt.Fprintf(os.Stderr, "Dial error: %s\n", dialError)
		return
	}

	// grab the auth status message that should be coming
	if message, err := expectMessage(connection); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return
	} else if message.Type != wrp.AuthMessageType {
		fmt.Fprintf(os.Stderr, "Expected auth status, but got: %s", message.Type)
		return
	} else if message.Status == nil {
		fmt.Fprintf(os.Stderr, "No auth status code")
		return
	} else if *message.Status != wrp.AuthStatusAuthorized {
		fmt.Fprintf(os.Stderr, "Expected authorized, but got: %d", *message.Status)
		return
	} else {
		fmt.Println("auth status received")
	}

	// spawn a go routine to respond to our routed message
	go func() {
		if message, err := expectMessage(connection); err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			return
		} else if message.Type != wrp.SimpleRequestResponseMessageType {
			fmt.Fprintf(os.Stderr, "Expected request/response, but got: %s", message.Type)
			return
		} else {
			fmt.Println("message received")
			deviceResponse := *message
			deviceResponse.Source = message.Destination
			deviceResponse.Destination = message.Source
			deviceResponse.Payload = []byte("Homer Simpson, Smiling Politely")

			if err := writeMessage(&deviceResponse, connection); err != nil {
				fmt.Fprintf(os.Stderr, "Unable to write response: %s", err)
				return
			}
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
		transactionComplete.Wait()
		fmt.Printf("(%s): %s to %s -> %s\n", response.Message.TransactionUUID, response.Message.Source, response.Message.Destination, response.Message.Payload)
	}

	// close the connection and ensure that the Disconnect event gets sent
	// before continuing.  This prevents a race condition with the example code
	// wrapper that swaps out the stream written to by the fmt.Print* functions.
	connection.Close()
	disconnected.Wait()

	// Output:
	// mac:111122223333 connected
	// auth status sent
	// auth status received
	// message sent
	// message received
	// response received
	// (MyTransactionUUID): mac:111122223333 to Example -> Homer Simpson, Smiling Politely
	// mac:111122223333 disconnected
}
