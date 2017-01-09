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
	CRUDCreateMessageType            = MessageType(5)
	CRUDRetrieveMessageType          = MessageType(6)
	CRUDUpdateMessageType            = MessageType(7)
	CRUDDeleteMessageType            = MessageType(8)
)

var (
	invalidMessageTypeString = "!!INVALID!!"

	messageTypeStrings = []string{
		invalidMessageTypeString,
		invalidMessageTypeString,
		"Auth",
		"SimpleRequestResponse",
		"SimpleEvent",
		"CRUDCreateMessageType",
		"CRUDRetrieveMessageType",
		"CRUDUpdateMessageType",
		"CRUDDeleteMessageType",
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
//
// TODO: Type isn't serialized as JSON right now.  If it can be serialized without
// breaking clients, that would simplify the Message mapping tags.
type Message struct {
	Type            MessageType             `msgpack:"msg_type" json:"-"`
	ContentType     string                  `msgpack:"content_type,omitempty" json:"content_type,omitempty"`
	TransactionUUID string                  `msgpack:"transaction_uuid,omitempty" json:"transaction_uuid,omitempty"`
	Source          string                  `msgpack:"source,omitempty" json:"source,omitempty"`
	Destination     string                  `msgpack:"dest,omitempty" json:"dest,omitempty"`
	Headers         []string                `msgpack:"headers,omitempty" json:"headers,omitempty"`
	Metadata        map[string]interface{}  `msgpack:"metadata,omitempty" json:"metadata,omitempty"`
	Spans           [][]interface{}         `msgpack:"spans,omitempty" json:"spans,omitempty"`
	IncludeSpans    bool                    `msgpack:"include_spans,omitempty" json:"include_spans,omitempty"`
	Status          *int64                  `msgpack:"status,omitempty" json:"status,omitempty"`
	Path            string                  `msgpack:"path,omitempty" json:"path,omitempty"`
	Payload         []byte                  `msgpack:"payload,omitempty" json:"payload,omitempty"`
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
		`{Type: %s, ContentType: %s, TransactionUUID: %s, Source: %s, Destination: %s, Headers: %v, Metadata: %v, Spans: %v, IncludeSpans: %v, Status: %s, Path: %s, Payload: %v}`,
		m.Type,
		m.ContentType,
		m.TransactionUUID,
		m.Source,
		m.Destination,
		fmt.Sprintf("%v", m.Headers),
		fmt.Sprintf("%v", m.Metadata),
		fmt.Sprintf("%v", m.Spans),
		fmt.Sprintf("%v", m.IncludeSpans),
		status,
		m.Path,
		m.Payload,
	)
}

// DeduceType examines the message fields other than Type to determine
// what the message type is, then sets that type on the message.
//
// This method is a bit of a hack.  It allows for formats like JSON where
// we don't deserialize the type from the message.
func (m *Message) DeduceType() error {
	if m.Status != nil {
		m.Type = AuthMessageType
	} else if len(m.TransactionUUID) > 0 {
		m.Type = SimpleRequestResponseMessageType
	} else if len(m.TransactionUUID) == 0 && m.Payload != nil {
		m.Type = SimpleEventMessageType
	} else if len(m.Path) > 0 {
		return fmt.Errorf("Unable to deduce which CRUD message type: %s", m)
//		m.Type = // todo: CRUD Message Type, this could be any of the 4
	} else {
		return fmt.Errorf("Unable to deduce type for message: %s", m)
	}

	return nil
}

// Valid performs a basic validation check on a given message
func (m *Message) Valid() error {
	missing := ""

	switch m.Type {
	case AuthMessageType:
		if m.Status == nil {
			return fmt.Errorf("Missing status for message type: %s", m.Type)
		}

	case SimpleRequestResponseMessageType:
		if len(m.TransactionUUID) == 0 {
			missing += fmt.Sprintf("Missing transaction id for message type: %s\n", m.Type)
		}
		if len(m.Source) == 0 {
			missing += fmt.Sprintf("Missing source for message type: %s\n", m.Type)
		}
		if len(m.Destination) == 0 {
			missing += fmt.Sprintf("Missing destination for message type: %s\n", m.Type)
		}
		if len(m.Payload) == 0 {
			missing += fmt.Sprintf("Missing payload for message type: %s\n", m.Type)
		}

	case SimpleEventMessageType:
		if len(m.Source) == 0 {
			missing += fmt.Sprintf("Missing source for message type: %s\n", m.Type)
		}
		if len(m.Destination) == 0 {
			missing += fmt.Sprintf("Missing destination for message type: %s\n", m.Type)
		}
		if len(m.Payload) == 0 {
			missing += fmt.Sprintf("Missing payload for message type: %s\n", m.Type)
		}
	
	case CRUDCreateMessageType,
	     CRUDRetrieveMessageType,
	     CRUDUpdateMessageType,
	     CRUDDeleteMessageType:
		if len(m.Source) == 0 {
			missing += fmt.Sprintf("Missing source for message type: %s\n", m.Type)
		}
		if len(m.Destination) == 0 {
			missing += fmt.Sprintf("Missing destination for message type: %s\n", m.Type)
		}
		if len(m.Path) == 0 {
			missing += fmt.Sprintf("Missing path for message type: %s\n", m.Type)
		}

	default:
		return fmt.Errorf("Invalid message type: %d", m.Type)
	}

	if missing != "" {
		return fmt.Errorf(missing)
	} else {
		return nil
	}
}

// NewAuth is a convenience factory function for creating
// an authorization WRP message
func NewAuth(status int64) *Message {
	return &Message{
		Type:   AuthMessageType,
		Status: &status,
	}
}

// NewSimpleRequestResponse is a convenience factory function for creating
// a simple request/response message
func NewSimpleRequestResponse(destination, source, uuid string, payload []byte) *Message {
	return &Message{
		Type:             SimpleRequestResponseMessageType,
		TransactionUUID:  uuid,
		Source:           source,
		Destination:      destination,
		Payload:          payload,
	}
}

// NewSimpleEvent is a convenience factory function for creating
// a simple event message
func NewSimpleEvent(destination, source string, payload []byte) *Message {
	return &Message{
		Type:        SimpleEventMessageType,
		Source:      source,
		Destination: destination,
		Payload:     payload,
	}
}

// NewCRUD is a convenience factory function for creating
// a CRUD message type
func newCRUD(crudType MessageType, destination, source, path string) *Message {
	return &Message{
		Type:            crudType,
		Source:          source,
		Destination:     destination,
		Path:            path,
	}
}

// NewCRUDCreate is a convenience factory function for creating
// a CRUD create message type
func NewCRUDCreate(destination, source, path string) *Message {
	return newCRUD(CRUDCreateMessageType, destination, source, path)
}

// NewCRUDRetrieve is a convenience factory function for creating
// a CRUD retrieve message type
func NewCRUDRetrieve(destination, source, path string) *Message {
	return newCRUD(CRUDRetrieveMessageType, destination, source, path)
}

// NewCRUDUpdate is a convenience factory function for creating
// a CRUD update message type
func NewCRUDUpdate(destination, source, path string) *Message {
	return newCRUD(CRUDUpdateMessageType, destination, source, path)
}

// NewCRUDDelete is a convenience factory function for creating
// a CRUD delete message type
func NewCRUDDelete(destination, source, path string) *Message {
	return newCRUD(CRUDDeleteMessageType, destination, source, path)
}