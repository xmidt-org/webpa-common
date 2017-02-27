package device

import (
	"bytes"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"io/ioutil"
	"os"
)

func ExampleManagerSimple() {
	options := &Options{
		Logger: &logging.LoggerWriter{ioutil.Discard},
		MessageReceivedListener: func(device Interface, message *wrp.Message, encoded []byte) {
			fmt.Printf("%s -> %s\n", message.Destination, message.Payload)
			err := device.Send(
				wrp.NewSimpleRequestResponse(message.Destination, message.Source, []byte("Homer Simpson, smiling politely")),
				nil,
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
	var (
		requestMessage = wrp.NewSimpleRequestResponse("destination.com", "somewhere.com", []byte("Billy Corgan, Smashing Pumpkins"))
		requestBuffer  bytes.Buffer
		encoder        = wrp.NewEncoder(&requestBuffer, wrp.Msgpack)
	)

	if err := encoder.Encode(requestMessage); err != nil {
		fmt.Printf("Unable to encode request: %s\n", err)
		return
	}

	if _, err := connection.Write(requestBuffer.Bytes()); err != nil {
		fmt.Printf("Unable to send request: %s\n", err)
		return
	}

	var (
		responseMessage wrp.Message
		responseBuffer  bytes.Buffer
		decoder         = wrp.NewDecoder(&responseBuffer, wrp.Msgpack)
	)

	if frameRead, err := connection.Read(&responseBuffer); err != nil {
		fmt.Printf("Unable to read response: %s\n", err)
		return
	} else if !frameRead {
		fmt.Println("Response frame skipped")
		return
	} else if err := decoder.Decode(&responseMessage); err != nil {
		fmt.Printf("Unable to decode response: %s\n", err)
		return
	}

	fmt.Printf("%s\n", responseMessage.Payload)

	// Output:
	// destination.com -> Billy Corgan, Smashing Pumpkins
	// Homer Simpson, smiling politely
}
