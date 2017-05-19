package wrp

import (
	"net/http"
	"strings"
	"fmt"
	"strconv"
)

// Constant HTTP header strings representing WRP fields
const (
	MsgTypeHeader = "X-Midt-Msg-Type"
	ContentTypeHeader = "X-Midt-Content-Type"
	AcceptHeader = "X-Midt-Accept"
	TrasactionUuidHeader = "X-Midt-Transaction-Uuid"
	StatusHeader = "X-Midt-Status"
	RDRHeader = "X-Midt-Request-Delivery-Reponse"
	HeadersArrHeader = "X-Midt-Headers"
	IncludeSpansHeader = "X-Midt-Include-Spans"
	SpansHeader = "X-Midt-Spans"
	CallTimeoutHeader = "X-Midt-Call-Timeout"
	PathHeader = "X-Midt-Path"
	SourceHeader = "X-Midt-Source"
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
	if !strings.EqualFold(msgType,"") && StringToMessageType(msgType) != MessageType(-1) {
			msg.Type = StringToMessageType(msgType)
	} else {
			return nil, fmt.Errorf("%s", "Invalid Message Type header string")
	}
	
	// all other fields are optional
	if contType := header.Get(ContentTypeHeader); !strings.EqualFold(contType,"") {
		msg.ContentType = contType
	}
	
	if accept := header.Get(AcceptHeader); !strings.EqualFold(accept,"") {
		msg.Accept = accept
	}
	
	if transUuid := header.Get(TrasactionUuidHeader); !strings.EqualFold(transUuid,"") {
		msg.TransactionUUID = transUuid
	}
	
	if status := header.Get(StatusHeader); !strings.EqualFold(status,"") {
		if statusInt, err := strconv.ParseInt(status, 10, 64); err != nil {
			msg.SetStatus(statusInt)
		}
	}
	
	if rdr := header.Get(RDRHeader); !strings.EqualFold(rdr,"") {
		if rdrInt, err := strconv.ParseInt(rdr, 10, 64); err != nil {
			msg.SetRequestDeliveryResponse(rdrInt)
		}
	}
	
	if path := header.Get(PathHeader); !strings.EqualFold(path,"") {
		msg.Path = path
	}
	
	if src := header.Get(SourceHeader); !strings.EqualFold(src,"") {
		msg.Source = src
	}
	
	if includeSpans := header.Get(IncludeSpansHeader); !strings.EqualFold(includeSpans,"") {
		if spansBool, err := strconv.ParseBool(includeSpans); err != nil {
			msg.SetIncludeSpans(spansBool)
		}
	}
	
	// TODO: Headers and Spans
	
	return msg, nil
}

