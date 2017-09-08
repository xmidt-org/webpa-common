package wrp

import (
	"fmt"
	"strconv"
)

//go:generate stringer -type=MessageType

// MessageType indicates the kind of WRP message
type MessageType int64

const (
	AuthorizationStatusMessageType MessageType = iota + 2
	SimpleRequestResponseMessageType
	SimpleEventMessageType
	CreateMessageType
	RetrieveMessageType
	UpdateMessageType
	DeleteMessageType
	ServiceRegistrationMessageType
	ServiceAliveMessageType
	lastMessageType

	AuthStatusAuthorized      = 200
	AuthStatusUnauthorized    = 401
	AuthStatusPaymentRequired = 402
	AuthStatusNotAcceptable   = 406
)

// SupportsTransaction tests if messages of this type are allowed to participate in transactions.
// If this method returns false, the TransactionUUID field should be ignored (but passed through
// where applicable).
func (mt MessageType) SupportsTransaction() bool {
	switch mt {
	case AuthorizationStatusMessageType:
		return false
	case SimpleEventMessageType:
		return false
	case ServiceRegistrationMessageType:
		return false
	case ServiceAliveMessageType:
		return false
	default:
		return true
	}
}

var (
	// stringToMessageType is a simple map of allowed strings which uniquely indicate MessageType values.
	// Included in this map are integral string keys.  Keys are assumed to be case insensitive.
	stringToMessageType map[string]MessageType
)

func init() {
	stringToMessageType = make(map[string]MessageType, lastMessageType-1)
	suffixLength := len("MessageType")

	// for each MessageType, allow the following string representations:
	//
	// The integral value of the constant
	// The String() value
	// The String() value minus the MessageType suffix
	for v := AuthorizationStatusMessageType; v < lastMessageType; v++ {
		stringToMessageType[strconv.Itoa(int(v))] = v

		vs := v.String()
		stringToMessageType[vs] = v
		stringToMessageType[vs[0:len(vs)-suffixLength]] = v
	}
}

// StringToMessageType converts a string into an enumerated MessageType constant.
// If the value equals the friendly name of a type, e.g. "Auth" for AuthMessageType,
// that type is returned.  Otherwise, the value is converted to an integer and looked up,
// with an error being returned in the event the integer value is not valid.
func StringToMessageType(value string) (MessageType, error) {
	mt, ok := stringToMessageType[value]
	if !ok {
		return MessageType(-1), fmt.Errorf("Invalid message type: %s", value)
	}

	return mt, nil
}
