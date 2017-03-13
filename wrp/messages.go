package wrp

// MessageType indicates the kind of WRP message
type MessageType int64

const (
	AuthMessageType                  = MessageType(2)
	SimpleRequestResponseMessageType = MessageType(3)
	SimpleEventMessageType           = MessageType(4)
	CreateMessageType                = MessageType(5)
	RetrieveMessageType              = MessageType(6)
	UpdateMessageType                = MessageType(7)
	DeleteMessageType                = MessageType(8)
	ServiceRegistrationMessageType   = MessageType(9)
	ServiceAliveMessageType          = MessageType(10)

	InvalidMessageTypeString = "!!INVALID!!"
)

func (mt MessageType) String() string {
	switch mt {
	case AuthMessageType:
		return "Auth"
	case SimpleRequestResponseMessageType:
		return "SimpleRequestResponse"
	case SimpleEventMessageType:
		return "SimpleEvent"
	case CreateMessageType:
		return "Create"
	case RetrieveMessageType:
		return "Retrieve"
	case UpdateMessageType:
		return "Update"
	case DeleteMessageType:
		return "Delete"
	case ServiceRegistrationMessageType:
		return "ServiceRegistration"
	case ServiceAliveMessageType:
		return "ServiceAlive"
	}

	return InvalidMessageTypeString
}

// EncoderTo describes the behavior of a message that can encode itself.
// Implementations of this interface will ensure that the MessageType is
// set correctly prior to encoding.
type EncoderTo interface {
	// EncodeTo encodes this message to the given Encoder
	EncodeTo(Encoder) error
}

// Routable describes an object which can be routed.  Implementations will most
// often also be WRP messages, and can be passed to decoders and encoders.
//
// Not all WRP messages are Routable.  Only messages that can be sent through
// routing software (e.g. talaria) implement this interface.
type Routable interface {
	// MessageType is the type of message represented by this Routable.
	MessageType() MessageType

	// To is the destination of this Routable instance.  It corresponds to the Destination field
	// in WRP messages defined in this package.
	To() string

	// From is the originator of this Routable instance.  It corresponds to the Source field
	// in WRP messages defined in this package.
	From() string
}

// Message is the union of all WRP fields, made optional (except for Type).  This type is
// useful for transcoding streams, since deserializing from non-msgpack formats like JSON
// has some undesireable side effects.
//
// IMPORTANT: Anytime a new WRP field is added to any message, or a new message with new fields,
// those new fields must be added to this struct for transcoding to work properly.  And of course:
// update the tests!
//
// For server code that sends specific messages, use one of the other WRP structs in this package.
//
// For server code that needs to read one format and emit another, use this struct as it allows
// client code to transcode without knowledge of the exact type of message.
type Message struct {
	Type                    MessageType       `wrp:"msg_type"`
	Source                  string            `wrp:"source,omitempty"`
	Destination             string            `wrp:"dest,omitempty"`
	TransactionUUID         string            `wrp:"transaction_uuid,omitempty"`
	ContentType             string            `wrp:"content_type,omitempty"`
	Accept                  string            `wrp:"accept,omitempty"`
	Status                  *int64            `wrp:"status,omitempty"`
	RequestDeliveryResponse *int64            `wrp:"rdr,omitempty"`
	Headers                 []string          `wrp:"headers,omitempty"`
	Metadata                map[string]string `wrp:"metadata,omitempty"`
	Spans                   [][]string        `wrp:"spans,omitempty"`
	IncludeSpans            *bool             `wrp:"include_spans,omitempty"`
	Path                    string            `wrp:"path,omitempty"`
	Objects                 string            `wrp:"objects,omitempty"`
	Payload                 []byte            `wrp:"payload,omitempty"`
	ServiceName             string            `wrp:"service_name,omitempty"`
	URL                     string            `wrp:"url,omitempty"`
}

// FailureResponse produces a Message which is the response to the failed delivery of this message.  If the response
// parameter is non-nil, that Message instance is used and is returned.  Otherwise, a new Message instance is created
// and returned.  Passing in this message as the response parameter is supported, but will mutate this message.
//
// The returned message will have the following changes from this message:
//
// (1) The Destination will be the Source of this message
// (2) The Source will be set to newSource
// (3) The RequestDeliveryResponse field will be set to requestDeliveryResponse
// (4) The Payload will be nil
//
// All other fields of the original message are copied over to the returned message.  If response was nil or
// points to a message other than this message, the returned message may be freely changed without affecting
// this message.
func (msg *Message) FailureResponse(response *Message, newSource string, requestDeliveryResponse int64) *Message {
	if response == nil {
		response = new(Message)
	}

	*response = *msg

	// set the destination first, to allow for response == msg
	response.Destination = msg.Source
	response.Source = newSource

	response.RequestDeliveryResponse = &requestDeliveryResponse
	response.Payload = nil

	return response
}

func (msg *Message) MessageType() MessageType {
	return msg.Type
}

func (msg *Message) To() string {
	return msg.Destination
}

func (msg *Message) From() string {
	return msg.Source
}

// SetStatus simplifies setting the optional Status field, which is a pointer type tagged with omitempty.
func (msg *Message) SetStatus(value int64) *Message {
	msg.Status = &value
	return msg
}

// SetRequestDeliveryResponse simplifies setting the optional RequestDeliveryResponse field, which is a pointer type tagged with omitempty.
func (msg *Message) SetRequestDeliveryResponse(value int64) *Message {
	msg.RequestDeliveryResponse = &value
	return msg
}

// SetIncludeSpans simplifies setting the optional IncludeSpans field, which is a pointer type tagged with omitempty.
func (msg *Message) SetIncludeSpans(value bool) *Message {
	msg.IncludeSpans = &value
	return msg
}

// AuthorizationStatus represents a WRP message of type AuthMessageType.
//
// https://github.com/Comcast/wrp-c/wiki/Web-Routing-Protocol#authorization-status-definition
type AuthorizationStatus struct {
	// Type is exposed principally for encoding.  This field *must* be set to AuthMessageType,
	// and is automatically set by the EncodeTo method.
	Type   MessageType `wrp:"msg_type"`
	Status int64       `wrp:"status"`
}

func (msg *AuthorizationStatus) EncodeTo(e Encoder) error {
	msg.Type = AuthMessageType
	return e.Encode(msg)
}

// SimpleRequestResponse represents a WRP message of type SimpleRequestResponseMessageType.
//
// https://github.com/Comcast/wrp-c/wiki/Web-Routing-Protocol#simple-request-response-definition
type SimpleRequestResponse struct {
	// Type is exposed principally for encoding.  This field *must* be set to SimpleRequestResponseMessageType,
	// and is automatically set by the EncodeTo method.
	Type                    MessageType       `wrp:"msg_type"`
	Source                  string            `wrp:"source"`
	Destination             string            `wrp:"dest"`
	ContentType             string            `wrp:"content_type,omitempty"`
	Accept                  string            `wrp:"accept,omitempty"`
	TransactionUUID         string            `wrp:"transaction_uuid,omitempty"`
	Status                  *int64            `wrp:"status,omitempty"`
	RequestDeliveryResponse *int64            `wrp:"rdr,omitempty"`
	Headers                 []string          `wrp:"headers,omitempty"`
	Metadata                map[string]string `wrp:"metadata,omitempty"`
	Spans                   [][]string        `wrp:"spans,omitempty"`
	IncludeSpans            *bool             `wrp:"include_spans,omitempty"`
	Payload                 []byte            `wrp:"payload,omitempty"`
}

// SetStatus simplifies setting the optional Status field, which is a pointer type tagged with omitempty.
func (msg *SimpleRequestResponse) SetStatus(value int64) *SimpleRequestResponse {
	msg.Status = &value
	return msg
}

// SetRequestDeliveryResponse simplifies setting the optional RequestDeliveryResponse field, which is a pointer type tagged with omitempty.
func (msg *SimpleRequestResponse) SetRequestDeliveryResponse(value int64) *SimpleRequestResponse {
	msg.RequestDeliveryResponse = &value
	return msg
}

// SetIncludeSpans simplifies setting the optional IncludeSpans field, which is a pointer type tagged with omitempty.
func (msg *SimpleRequestResponse) SetIncludeSpans(value bool) *SimpleRequestResponse {
	msg.IncludeSpans = &value
	return msg
}

func (msg *SimpleRequestResponse) EncodeTo(e Encoder) error {
	msg.Type = SimpleRequestResponseMessageType
	return e.Encode(msg)
}

func (msg *SimpleRequestResponse) MessageType() MessageType {
	return msg.Type
}

func (msg *SimpleRequestResponse) To() string {
	return msg.Destination
}

func (msg *SimpleRequestResponse) From() string {
	return msg.Source
}

// SimpleEvent represents a WRP message of type SimpleEventMessageType.
//
// https://github.com/Comcast/wrp-c/wiki/Web-Routing-Protocol#simple-event-definition
type SimpleEvent struct {
	// Type is exposed principally for encoding.  This field *must* be set to SimpleEventMessageType,
	// and is automatically set by the EncodeTo method.
	Type        MessageType       `wrp:"msg_type"`
	Source      string            `wrp:"source"`
	Destination string            `wrp:"dest"`
	ContentType string            `wrp:"content_type,omitempty"`
	Headers     []string          `wrp:"headers,omitempty"`
	Metadata    map[string]string `wrp:"metadata,omitempty"`
	Payload     []byte            `wrp:"payload,omitempty"`
}

func (msg *SimpleEvent) EncodeTo(e Encoder) error {
	msg.Type = SimpleEventMessageType
	return e.Encode(msg)
}

func (msg *SimpleEvent) MessageType() MessageType {
	return msg.Type
}

func (msg *SimpleEvent) To() string {
	return msg.Destination
}

func (msg *SimpleEvent) From() string {
	return msg.Source
}

// CRUD represents a WRP message of one of the CRUD message types.  This type does not implement EncodeTo,
// and so does not automatically set the Type field.  Client code must set the Type code appropriately.
//
// https://github.com/Comcast/wrp-c/wiki/Web-Routing-Protocol#crud-message-definition
type CRUD struct {
	Type                    MessageType       `wrp:"msg_type"`
	Source                  string            `wrp:"source"`
	Destination             string            `wrp:"dest"`
	TransactionUUID         string            `wrp:"transaction_uuid,omitempty"`
	Headers                 []string          `wrp:"headers,omitempty"`
	Metadata                map[string]string `wrp:"metadata,omitempty"`
	Spans                   [][]string        `wrp:"spans,omitempty"`
	IncludeSpans            *bool             `wrp:"include_spans,omitempty"`
	Status                  *int64            `wrp:"status,omitempty"`
	RequestDeliveryResponse *int64            `wrp:"rdr,omitempty"`
	Path                    string            `wrp:"path"`
	Objects                 string            `wrp:"objects,omitempty"`
}

// SetStatus simplifies setting the optional Status field, which is a pointer type tagged with omitempty.
func (msg *CRUD) SetStatus(value int64) *CRUD {
	msg.Status = &value
	return msg
}

// SetRequestDeliveryResponse simplifies setting the optional RequestDeliveryResponse field, which is a pointer type tagged with omitempty.
func (msg *CRUD) SetRequestDeliveryResponse(value int64) *CRUD {
	msg.RequestDeliveryResponse = &value
	return msg
}

// SetIncludeSpans simplifies setting the optional IncludeSpans field, which is a pointer type tagged with omitempty.
func (msg *CRUD) SetIncludeSpans(value bool) *CRUD {
	msg.IncludeSpans = &value
	return msg
}

func (msg *CRUD) MessageType() MessageType {
	return msg.Type
}

func (msg *CRUD) To() string {
	return msg.Destination
}

func (msg *CRUD) From() string {
	return msg.Source
}

// ServiceRegistration represents a WRP message of type ServiceRegistrationMessageType.
//
// https://github.com/Comcast/wrp-c/wiki/Web-Routing-Protocol#on-device-service-registration-message-definition
type ServiceRegistration struct {
	// Type is exposed principally for encoding.  This field *must* be set to ServiceRegistrationMessageType,
	// and is automatically set by the EncodeTo method.
	Type        MessageType `wrp:"msg_type"`
	ServiceName string      `wrp:"service_name"`
	URL         string      `wrp:"url"`
}

func (msg *ServiceRegistration) EncodeTo(e Encoder) error {
	msg.Type = ServiceRegistrationMessageType
	return e.Encode(msg)
}

// ServiceAlive represents a WRP message of type ServiceAliveMessageType.
//
// https://github.com/Comcast/wrp-c/wiki/Web-Routing-Protocol#on-device-service-alive-message-definition
type ServiceAlive struct {
	// Type is exposed principally for encoding.  This field *must* be set to ServiceAliveMessageType,
	// and is automatically set by the EncodeTo method.
	Type MessageType `wrp:"msg_type"`
}

func (msg *ServiceAlive) EncodeTo(e Encoder) error {
	msg.Type = ServiceAliveMessageType
	return e.Encode(msg)
}
