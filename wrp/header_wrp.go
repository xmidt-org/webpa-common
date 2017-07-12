package wrp

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Constant HTTP header strings representing WRP fields
const (
	MsgTypeHeader         = "X-Midt-Msg-Type"
	ContentTypeHeader     = "X-Midt-Content-Type"
	AcceptHeader          = "X-Midt-Accept"
	TransactionUuidHeader = "X-Midt-Transaction-Uuid"
	StatusHeader          = "X-Midt-Status"
	RDRHeader             = "X-Midt-Request-Delivery-Reponse"
	HeadersArrHeader      = "X-Midt-Headers"
	IncludeSpansHeader    = "X-Midt-Include-Spans"
	SpansHeader           = "X-Midt-Spans"
	PathHeader            = "X-Midt-Path"
	SourceHeader          = "X-Midt-Source"

	ErrInvalidMsgType         = "Invalid Message Type"
	ErrInvalidSource          = "Invalid Source"
	ErrInvalidTransactionUuid = "Invalid TransactionUuid"
	ErrInvalidStatus          = "Invalid Status"
	ErrInvalidPath            = "Invalid Path"
)

// Map string to MessageType int
func StringToMessageType(str string) MessageType {
	switch str {
	case "Auth":
		return AuthMessageType
	case "SimpleRequestResponse":
		return SimpleRequestResponseMessageType
	case "SimpleEvent":
		return SimpleEventMessageType
	case "Create":
		return CreateMessageType
	case "Retrieve":
		return RetrieveMessageType
	case "Update":
		return UpdateMessageType
	case "Delete":
		return DeleteMessageType
	case "ServiceRegistration":
		return ServiceRegistrationMessageType
	case "ServiceAlive":
		return ServiceAliveMessageType
	default:
		return -1
	}
}

// Convert HTTP header to WRP generic Message
func HeaderToWRP(header http.Header) (*Message, error) {
	msg := new(Message)

	// MessageType is mandatory
	msgType := header.Get(MsgTypeHeader)
	if !strings.EqualFold(msgType, "") && StringToMessageType(msgType) != MessageType(-1) {
		msg.Type = StringToMessageType(msgType)
	} else {
		return nil, fmt.Errorf("%s", ErrInvalidMsgType)
	}

	// Source is mandatory for SimpleRequestResponse, SimpleEvent and CRUD
	if src := header.Get(SourceHeader); !strings.EqualFold(src, "") {
		msg.Source = src
	} else if msg.Type == SimpleRequestResponseMessageType || msg.Type == SimpleEventMessageType ||
		msg.Type == CreateMessageType || msg.Type == RetrieveMessageType || msg.Type == UpdateMessageType ||
		msg.Type == DeleteMessageType {
		return nil, fmt.Errorf("%s", ErrInvalidSource)
	}

	// TransactionUuid is mandatory for SimpleRequestResponse and CRUD
	if transUuid := header.Get(TransactionUuidHeader); !strings.EqualFold(transUuid, "") {
		msg.TransactionUUID = transUuid
	} else if msg.Type == SimpleRequestResponseMessageType ||
		msg.Type == CreateMessageType || msg.Type == RetrieveMessageType || msg.Type == UpdateMessageType ||
		msg.Type == DeleteMessageType {
		return nil, fmt.Errorf("%s", ErrInvalidTransactionUuid)
	}

	// all other fields are optional
	if contType := header.Get(ContentTypeHeader); !strings.EqualFold(contType, "") {
		msg.ContentType = contType
	}

	if accept := header.Get(AcceptHeader); !strings.EqualFold(accept, "") {
		msg.Accept = accept
	}

	if status := header.Get(StatusHeader); !strings.EqualFold(status, "") {
		if statusInt, err := strconv.ParseInt(status, 10, 64); err == nil {
			msg.SetStatus(statusInt)
		} else {
			return nil, err
		}
	} else if msg.Type == AuthMessageType {
		return nil, fmt.Errorf("%s", ErrInvalidStatus)
	}

	if rdr := header.Get(RDRHeader); !strings.EqualFold(rdr, "") {
		if rdrInt, err := strconv.ParseInt(rdr, 10, 64); err == nil {
			msg.SetRequestDeliveryResponse(rdrInt)
		} else {
			return nil, err
		}
	}

	// path is mandatory for CRUD
	if path := header.Get(PathHeader); !strings.EqualFold(path, "") {
		msg.Path = path
	} else if msg.Type == CreateMessageType || msg.Type == RetrieveMessageType ||
		msg.Type == UpdateMessageType || msg.Type == DeleteMessageType {
		return nil, fmt.Errorf("%s", ErrInvalidPath)
	}

	if includeSpans := header.Get(IncludeSpansHeader); !strings.EqualFold(includeSpans, "") {
		if spansBool, err := strconv.ParseBool(includeSpans); err == nil {
			msg.SetIncludeSpans(spansBool)
		}
	}

	// Handle Headers and Spans which contain multiple values
	for key, value := range header {
		if strings.EqualFold(key, HeadersArrHeader) {
			if msg.Headers == nil {
				msg.Headers = []string{}
			}
			for item := range value {
				msg.Headers = append(msg.Headers, value[item])
			}
		}

		// Each span element will look like this {"name" , "start_time" , "duration"}
		if strings.EqualFold(key, SpansHeader) {
			if msg.Spans == nil {
				msg.Spans = make([][]string, len(value))
			}

			j := 0
			for i := 0; i < len(value); i++ {
				msg.Spans[j] = append(msg.Spans[j], value[i])
				if (i+1)%3 == 0 {
					j++
				}
			}
		}
	}

	return msg, nil
}

// Convert WRP generic Message to HTTP header
func WRPToHeader(msg *Message) (header http.Header, err error) {

	header = make(map[string][]string)

	if strings.EqualFold(msg.Type.String(), InvalidMessageTypeString) {
		return nil, fmt.Errorf("%s", ErrInvalidMsgType)
	} else {
		header.Add(MsgTypeHeader, msg.Type.String())
	}

	// Status is mandatory for AuthMessageType
	if msg.Status != nil && *msg.Status > int64(0) {
		header.Add(StatusHeader, strconv.FormatInt(*msg.Status, 10))
	} else if msg.Type == AuthMessageType {
		return nil, fmt.Errorf("%s", ErrInvalidStatus)
	}

	// Source is mandatory for SimpleRequestResponse, SimpleEvent and CRUD
	if !strings.EqualFold(msg.Source, "") {
		header.Add(SourceHeader, msg.Source)
	} else if msg.Type == SimpleRequestResponseMessageType ||
		msg.Type == SimpleEventMessageType || msg.Type == CreateMessageType ||
		msg.Type == RetrieveMessageType || msg.Type == UpdateMessageType || msg.Type == DeleteMessageType {
		return nil, fmt.Errorf("%s", ErrInvalidSource)
	}

	// TransactionUuid is mandatory for SimpleRequestResponse and CRUD
	if !strings.EqualFold(msg.TransactionUUID, "") {
		header.Add(TransactionUuidHeader, msg.TransactionUUID)
	} else if msg.Type == SimpleRequestResponseMessageType || msg.Type == CreateMessageType ||
		msg.Type == RetrieveMessageType || msg.Type == UpdateMessageType || msg.Type == DeleteMessageType {
		return nil, fmt.Errorf("%s", ErrInvalidTransactionUuid)
	}

	if !strings.EqualFold(msg.ContentType, "") {
		header.Add(ContentTypeHeader, msg.ContentType)
	}

	if !strings.EqualFold(msg.Accept, "") {
		header.Add(AcceptHeader, msg.Accept)
	}

	// path is mandatory for CRUD
	if !strings.EqualFold(msg.Path, "") {
		header.Add(PathHeader, msg.Path)
	} else if msg.Type == CreateMessageType || msg.Type == RetrieveMessageType ||
		msg.Type == UpdateMessageType || msg.Type == DeleteMessageType {
		return nil, fmt.Errorf("%s", "Invalid Path")
	}

	if msg.RequestDeliveryResponse != nil && *msg.RequestDeliveryResponse > int64(0) {
		header.Add(RDRHeader, strconv.FormatInt(*msg.RequestDeliveryResponse, 10))
	}

	if msg.IncludeSpans != nil && *msg.IncludeSpans == true {

		header.Add(IncludeSpansHeader, strconv.FormatBool(*msg.IncludeSpans))

		if msg.Spans != nil {
			for i := 0; i < len(msg.Spans); i++ {
				for _, span := range msg.Spans[i] {
					header.Add(SpansHeader, span)
				}
			}
		}
	} else if msg.IncludeSpans != nil && *msg.IncludeSpans == false {
		header.Add(IncludeSpansHeader, strconv.FormatBool(*msg.IncludeSpans))
	}

	if msg.Headers != nil {
		if msg.Headers != nil {
			for _, hdr := range msg.Headers {
				header.Add(HeadersArrHeader, hdr)
			}
		}
	}

	return
}
