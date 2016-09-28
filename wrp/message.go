package wrp

import (
	"fmt"
	"strconv"
)

// MessageType indicates the kind of WRP message
type MessageType int64

const (
	AuthMessageType                  = MessageType(2)
	SimpleRequestResponseMessageType = MessageType(3)
	SimpleEventMessageType           = MessageType(4)
)

var (
	invalidMessageTypeString = "!!INVALID!!"

	messageTypeStrings = []string{
		invalidMessageTypeString,
		invalidMessageTypeString,
		"Auth",
		"SimpleRequestResponse",
		"SimpleEvent",
	}
)

func (mt MessageType) String() string {
	if int(mt) < len(messageTypeStrings) {
		return messageTypeStrings[mt]
	}

	return invalidMessageTypeString
}

// Message represents a single WRP message.  The Type field determines how the other fields
// are interpreted.  For example, if the Type is AuthMessageType, then only Status will be set.
type Message struct {
	Type            MessageType `msgpack:"msg_type" json:"-"`
	Status          *int64      `msgpack:"status,omitempty" json:"status,omitempty"`
	TransactionUUID string      `msgpack:"transaction_uuid,omitempty" json:"transaction_uuid,omitempty"`
	Source          string      `msgpack:"source,omitempty" json:"source,omitempty"`
	Destination     string      `msgpack:"dest,omitempty" json:"dest,omitempty"`
	Payload         []byte      `msgpack:"payload,omitempty" json:"payload,omitempty"`
}

// String returns a useful string representation of this message
func (m *Message) String() string {
	if m == nil {
		return "nil"
	}

	status := "nil"
	if m.Status != nil {
		status = strconv.FormatInt(*m.Status, 10)
	}

	return fmt.Sprintf(
		`{Type: %s, Status: %s, Source: %s, Destination: %s, Payload: %v}`,
		m.Type,
		status,
		m.Source,
		m.Destination,
		m.Payload,
	)
}

// Valid performs a basic validation check on a given message
func (m *Message) Valid() error {
	switch m.Type {
	case AuthMessageType:
		if m.Status == nil {
			return fmt.Errorf("Missing status for message type: %s", m.Type)
		}

	case SimpleRequestResponseMessageType:
		fallthrough

	case SimpleEventMessageType:
		if len(m.Destination) == 0 {
			return fmt.Errorf("Missing destination for message type: %s", m.Type)
		}

	default:
		return fmt.Errorf("Invalid message type: %d", m.Type)
	}

	return nil
}
