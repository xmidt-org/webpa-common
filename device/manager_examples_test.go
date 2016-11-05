package device

import (
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"io/ioutil"
	"os"
)

func ExampleManagerSimple() {
	logger := &logging.LoggerWriter{ioutil.Discard}

	options := &Options{
		Logger: logger,
		MessageListener: func(device Interface, message *wrp.Message) {
			fmt.Printf("%s -> %s\n", message.Destination, message.Payload)
			err := device.Send(
				wrp.NewSimpleRequestResponse(message.Destination, message.Source, []byte("Homer Simpson, smiling politely")),
			)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to send response: %s", err)
			}
		},
	}

	_, server, websocketURL := startWebsocketServer(options)
	defer server.Close()

	dialer := NewDialer(options, nil)
	connection, _, err := dialer.Dial(
		websocketURL,
		"mac:111122223333",
		nil,
		nil,
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to server: %s\n", err)
		return
	}

	defer connection.Close()

	requestMessage := wrp.NewSimpleRequestResponse("destination.com", "somewhere.com", []byte("Billy Corgan, Smashing Pumpkins"))
	if err := connection.Write(requestMessage); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to send event: %s", err)
		return
	}

	responseMessage, err := connection.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read response: %s", err)
		return
	}

	fmt.Printf("%s\n", responseMessage.Payload)

	// Output:
	// destination.com -> Billy Corgan, Smashing Pumpkins
	// Homer Simpson, smiling politely
}
