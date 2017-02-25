package device

import (
	"github.com/Comcast/webpa-common/wrp"
)

// Envelope is the message type received by devices serviced by a Router.
// An envelope wraps a wrp.Message and provides routing information (e.g. the device ID).
type Envelope struct {
	// ID is the device ID to which this message should be sent
	ID ID

	// Message is the wrp Message to encode and send to the device
	Message wrp.Message

	// Encoded is the pre-encoded message to send to the device.  This is optional, and
	// if supplied will be used instead of (possibly) re-encoding the Message.
	Encoded []byte
}

// DecodeEnvelope extracts a WRP message from the given Decoder and wraps it
// in an envelope with routing information.
func DecodeEnvelope(decoder wrp.Decoder) (envelope *Envelope, err error) {
	envelope = new(Envelope)
	err = decoder.Decode(&envelope.Message)
	if err != nil {
		return
	}

	envelope.ID, err = ParseID(envelope.Message.Destination)
	return
}
